#import <AppKit/AppKit.h>
#import <objc/runtime.h>

static void (*g_urlCallback)(const char *);

// bcOpenURLs is the NSApplicationDelegate method added to glfw's delegate
// class. macOS calls it when this app is launched for an http(s) link, so it
// forwards each URL to Go.
static void bcOpenURLs(id self, SEL _cmd, NSApplication *app, NSArray *urls) {
    (void)self; (void)_cmd; (void)app;
    if (!g_urlCallback) return;
    for (NSURL *url in urls) {
        const char *s = [[url absoluteString] UTF8String];
        g_urlCallback(s ? s : "");
    }
}

// installURLReceiver adds application:openURLs: to the app's current
// NSApplication delegate class, so the URL arrives without replacing glfw's
// delegate (which owns window/quit lifecycle).
void installURLReceiver(void (*cb)(const char *)) {
    NSApplication *app = [NSApplication sharedApplication];
    if (app == nil) return;
    g_urlCallback = cb;
    id delegate = [app delegate];
    if (delegate == nil) return;
    if ([delegate respondsToSelector:@selector(application:openURLs:)]) {
        return; // a handler is already installed
    }
    class_addMethod([delegate class], @selector(application:openURLs:),
                    (IMP)bcOpenURLs, "v@:@@");
}
