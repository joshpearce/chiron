#if targetEnvironment(macCatalyst)
import UIKit

/// "Send to Chiron" in the Services menu of every app on the Mac: the
/// selected text becomes a capture and Chiron comes forward on it, the
/// way a share from the iPad does. Services are AppKit's, which a
/// Catalyst app does not link, so the provider is registered and the
/// pasteboard read by name through the runtime; the entry itself is the
/// NSServices item in Info.plist.
enum MacServices {
    private static let provider = Provider()

    static func register() {
        application?.setValue(provider, forKey: "servicesProvider")
    }

    /// AppKit's NSApp, by name.
    private static var application: NSObject? {
        guard let cls = NSClassFromString("NSApplication") as? NSObject.Type else { return nil }
        return cls.perform(NSSelectorFromString("sharedApplication"))?.takeUnretainedValue() as? NSObject
    }

    /// The app in front of the one the selection was made in. Opening the
    /// capture's URL alone delivers it but leaves the window behind, and a
    /// sheet waits for the scene to come forward. Since macOS 14 an app in
    /// the background may not simply activate itself; it takes activation
    /// over from the app in front, which is what a service provider does.
    static func activate() {
        guard let running = NSClassFromString("NSRunningApplication") as? NSObject.Type,
              let workspace = NSClassFromString("NSWorkspace") as? NSObject.Type,
              let me = running.perform(NSSelectorFromString("currentApplication"))?.takeUnretainedValue() as? NSObject,
              let shared = workspace.perform(NSSelectorFromString("sharedWorkspace"))?.takeUnretainedValue() as? NSObject,
              let front = shared.value(forKey: "frontmostApplication") as? NSObject else { return }
        let selector = NSSelectorFromString("activateFromApplication:options:")
        guard let method = me.method(for: selector) else { return }
        typealias Activate = @convention(c) (AnyObject, Selector, AnyObject, UInt) -> Bool
        let ignoringOtherApps: UInt = 1 << 1
        _ = unsafeBitCast(method, to: Activate.self)(me, selector, front, ignoringOtherApps)
    }

    final class Provider: NSObject {
        /// The service's entry point: `sendToChiron:userData:error:`.
        @objc func sendToChiron(_ pasteboard: NSObject, userData: String, error: AutoreleasingUnsafeMutablePointer<NSString?>) {
            guard let id = capture(from: pasteboard) else {
                error.pointee = "Nothing Chiron can read was selected."
                return
            }
            DispatchQueue.main.async {
                MacServices.activate()
                UIApplication.shared.open(CaptureInbox.url(for: id))
            }
        }

        /// The text on the pasteboard, kept in the inbox; nil when there is
        /// nothing to keep.
        func capture(from pasteboard: NSObject) -> String? {
            let text = pasteboard.perform(NSSelectorFromString("stringForType:"), with: "public.utf8-plain-text")?
                .takeUnretainedValue() as? String ?? ""
            let c = Capture(text: text.trimmingCharacters(in: .whitespacesAndNewlines))
            guard !c.isEmpty, (try? CaptureInbox.write(c)) != nil else { return nil }
            return c.id
        }
    }
}
#endif
