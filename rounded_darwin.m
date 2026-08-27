#import <AppKit/AppKit.h>
#import <QuartzCore/QuartzCore.h>

// Round the corners of a borderless NSWindow by making it transparent and
// clipping its content view to a rounded rect, then letting macOS draw a
// shadow that follows that shape.
void roundWindow(void *win) {
    NSWindow *w = (NSWindow *)win;
    if (w == nil) return;
    [w setOpaque:NO];
    [w setBackgroundColor:[NSColor clearColor]];
    [w setHasShadow:YES];
    NSView *cv = [w contentView];
    [cv setWantsLayer:YES];
    CALayer *layer = [cv layer];
    [layer setCornerRadius:16.0];
    [layer setMasksToBounds:YES];
}
