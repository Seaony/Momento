import AppKit
import Foundation

final class AppOpenHandler: NSObject, NSApplicationDelegate {
    var onOpenLibraryURLs: (([URL]) -> Bool)?
    private var pendingOpenURLs: [URL] = []

    func application(_ sender: NSApplication, openFiles filenames: [String]) {
        let urls = filenames.map(URL.init(fileURLWithPath:))

        guard !urls.isEmpty else {
            sender.reply(toOpenOrPrint: .failure)
            return
        }

        if let onOpenLibraryURLs {
            sender.reply(toOpenOrPrint: onOpenLibraryURLs(urls) ? .success : .failure)
        } else {
            pendingOpenURLs.append(contentsOf: urls)
            sender.reply(toOpenOrPrint: .success)
        }
    }

    func flushPendingLibraryURLs() {
        guard !pendingOpenURLs.isEmpty, let onOpenLibraryURLs else {
            return
        }

        let urls = pendingOpenURLs
        pendingOpenURLs.removeAll()
        _ = onOpenLibraryURLs(urls)
    }
}
