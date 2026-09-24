import Foundation
import UserNotifications

/// A word to the reader when the server finishes something they asked
/// for and may have put the iPad down waiting on: a primer or a book
/// written, a change request built or failed.
protocol NoticeSender: AnyObject {
    /// Ask the system, once, whether notices may be shown. Called when the
    /// reader starts something worth telling them about, so the question
    /// comes when it makes sense.
    func prepare()
    /// Show a notice now; the same id replaces an earlier one.
    func notify(id: String, title: String, body: String)
}

final class Notices: NSObject, NoticeSender, UNUserNotificationCenterDelegate {
    static let shared = Notices()

    private var center: UNUserNotificationCenter {
        let c = UNUserNotificationCenter.current()
        c.delegate = self
        return c
    }

    func prepare() {
        center.requestAuthorization(options: [.alert, .sound]) { _, _ in }
    }

    func notify(id: String, title: String, body: String) {
        let content = UNMutableNotificationContent()
        content.title = title
        content.body = body
        content.sound = .default
        center.add(UNNotificationRequest(identifier: id, content: content, trigger: nil))
    }

    /// The notice shows as a banner even while Chiron is in front: the
    /// reader may be in a book, or the app may be beside another.
    func userNotificationCenter(_ center: UNUserNotificationCenter, willPresent notification: UNNotification) async -> UNNotificationPresentationOptions {
        [.banner, .sound]
    }
}
