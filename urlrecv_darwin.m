#import <AppKit/AppKit.h>
#import <objc/runtime.h>
#include "_cgo_export.h"

// bcOpenURLs is the NSApplicationDelegate method added to glfw's delegate
// class. macOS calls it when this app is launched for an http(s) link, so it
// forwards each URL to the Go onIncomingURL export (declared in the generated
// _cgo_export.h rather than passed as a function pointer, which cgo can't
// resolve as C.<name> on all toolchains).
static void bcOpenURLs(id self, SEL _cmd, NSApplication *app, NSArray *urls) {
    (void)self; (void)_cmd; (void)app;
    for (NSURL *url in urls) {
        const char *s = [[url absoluteString] UTF8String];
        onIncomingURL((char *)(s ? s : ""));
    }
}

// installURLReceiver adds application:openURLs: to the app's current
// NSApplication delegate class, so the URL arrives without replacing glfw's
// delegate (which owns window/quit lifecycle).
void installURLReceiver(void) {
    NSApplication *app = [NSApplication sharedApplication];
    if (app == nil) return;
    id delegate = [app delegate];
    if (delegate == nil) return;
    if ([delegate respondsToSelector:@selector(application:openURLs:)]) {
        return; // a handler is already installed
    }
    class_addMethod([delegate class], @selector(application:openURLs:),
                    (IMP)bcOpenURLs, "v@:@@");
}
