# iPad reading/tutoring app on iOS 15: research report

Companion to `IPAD-PLAN.md` (repo root). Gathered 2026-09-01 against Apple's
documentation endpoints, Apple's simulator runtime index, community sources,
and the local toolchain (Xcode 26.6 build 17F113, macOS 26.5.2, only the
iOS 26.5 runtime installed). Items marked UNVERIFIED come from secondary
sources or memory. The test device is Matt's iPad mini 4 (iPad5,1, iPadOS
15.8.8) - see `ipad-app/project.yml`.

## 0. Two flags before anything else

- **The target iPad cannot use Apple Pencil.** The only iPads whose last OS
  is iPadOS 15.8.x are the iPad Air 2 and iPad mini 4
  (https://endoflife.date/ipad, https://en.wikipedia.org/wiki/IOS_15).
  Neither supports any Apple Pencil generation
  (https://support.apple.com/en-us/108937). Scribble also needs a Pencil.
  "Handwritten answers" on this device means a finger or a capacitive
  stylus, which the system sees as a finger. Plan PencilKit with
  `drawingPolicy = .anyInput` in a dedicated write-box, never pencil-only
  ink over scrolling text.
- **Stage Manager is iPadOS 16.1+ and needs newer hardware.** Ignore it.
  Split View and Slide Over do apply.

## 1. HIG for iPadOS (reading/learning app)

**Layout, safe areas, readable width**
- Respect safe areas and layout guides; the system guides "apply standard
  margins around content and restrict the width of text for optimal
  readability" (https://developer.apple.com/design/human-interface-guidelines/layout).
  UIKit exposes `readableContentGuide`, "an area that can easily be read
  without forcing users to move their head to track the lines"
  (https://developer.apple.com/documentation/uikit/uiview/readablecontentguide).
  SwiftUI has no equivalent on iOS 15; cap the text column with
  `.frame(maxWidth: ~680)` and center it.
- Size classes: a 9.7"/7.9" iPad is regular x regular full screen. Under
  Split View on non-Pro iPads both apps are compact-width except the
  primary app in the landscape 2/3 slot; Slide Over is always compact
  (https://developer.apple.com/library/archive/documentation/WindowsViews/Conceptual/AdoptingMultitaskingOniPad/QuickStartForSlideOverAndSplitView.html).
  Eligibility requires a launch screen and all four orientations;
  `UIRequiresFullScreen = YES` opts out.
- HIG: "every app needs to work well with multitasking"; apps must "save
  and restore their context"; apps "don't control multitasking
  configurations or receive any indication"
  (https://developer.apple.com/design/human-interface-guidelines/multitasking).
  Design the reader for a 320pt-wide compact column as the floor.

**Sidebars vs tab bars, navigation**
- "Consider using a tab bar first"; "show no more than two levels of
  hierarchy in a sidebar"; let people hide the sidebar with the edge swipe
  (https://developer.apple.com/design/human-interface-guidelines/sidebars).
  The top-of-screen iPad tab bar is iPadOS 18
  (https://developer.apple.com/design/human-interface-guidelines/tab-bars);
  on iOS 15 a tab bar is at the bottom. Suggested shape: bookshelf -> book
  (contents as sidebar in regular width, pushed list in compact) -> chapter
  reader full-bleed.

**Typography and Dynamic Type**
- iPadOS body text style is 17pt; minimum 11pt; "avoid light font weights";
  use text styles so Dynamic Type works; "for wide columns or long passages,
  more space between lines"; reduce columns as font size grows
  (https://developer.apple.com/design/human-interface-guidelines/typography).
  New York (serif) is available to apps. Accessibility: allow text
  enlargement of at least 200 percent
  (https://developer.apple.com/design/human-interface-guidelines/accessibility).

**Dark mode and contrast**
- Contrast at least 4.5:1, aim for 7:1 for custom colors and small text;
  semantic system colors; test with Increase Contrast and Reduce Transparency
  (https://developer.apple.com/design/human-interface-guidelines/dark-mode).
  Reading apps add sepia/paper themes; keep them explicit themes, not
  overrides of system semantics.

**Keyboard and pointer**
- Respect standard shortcuts; holding Command shows the shortcut HUD
  (https://developer.apple.com/design/human-interface-guidelines/keyboards).
  SwiftUI `.keyboardShortcut()` is iOS 14+. Reader shortcuts: arrows/space
  page, Cmd-] / Cmd-[ chapter, number keys pick a choice. UIKit
  `keyboardLayoutGuide` is iOS 15+; SwiftUI keyboard avoidance is automatic
  on iOS 14+ (UNVERIFIED edge cases inside custom scroll views).
- Pointer: highlight/lift/hover effects; I-beam over text automatically
  (https://developer.apple.com/design/human-interface-guidelines/pointing-devices).
  SwiftUI `.hoverEffect()` is iOS 13.4+.

**Apple Pencil and Scribble (only on Pencil-capable iPads)**
- Scribble "works in all standard text components - text fields, text
  views, search fields, editable fields in web content - except password
  fields"; keep the field stationary; enlarge the field; double-tap must not
  perform content-modifying actions
  (https://developer.apple.com/design/human-interface-guidelines/apple-pencil-and-scribble).

**Patterns from the best reading apps**
- Apple Books: Curl / Fast Fade / Scroll page modes, font, size, theme,
  brightness in one Customize sheet
  (https://support.apple.com/guide/ipad/read-books-ipadc8494b6b/ipados).
  Tap-center for chrome, tap/swipe edges to page is the Books/Kindle
  convention (UNVERIFIED exact tap-zone specs).
- Notability/Goodnotes: a zoomed "writing box" for small handwriting
  (https://support.gingerlabs.com/hc/en-us/articles/218333197-Writing-with-Apple-Pencil).
  The writing box maps well to handwritten-answer boxes.

## 2. SwiftUI vs UIKit on iOS 15; toolchain facts

**Toolchain (verified locally and in Apple release notes)**
- Xcode 26 supports on-device debugging on iOS 15+ and requires macOS 15.6+
  (https://developer.apple.com/documentation/xcode-release-notes/xcode-26-release-notes).
  Apple staff: "The Deployment Target in Xcode is aligned with simulator
  support and on-device support"; "no support for simulators before iOS
  15.0" (https://developer.apple.com/forums/thread/805506). iOS 15 is the
  floor for both target and simulator in Xcode 26. UNVERIFIED whether Xcode
  27 keeps iOS 15.
- **iOS 15.5 simulator is installable on this Mac.** Apple's runtime index
  (https://devimages-cdn.apple.com/downloads/xcode/simulators/index2.dvtdownloadableindex)
  lists iOS 15.0, 15.2, 15.4, 15.5 simulators with `maxXcodeVersion
  26.99.0`; iOS 16.x is also capped at 26.99, suggesting Xcode 27 drops
  both. Command (flag confirmed in `xcodebuild -help` on Xcode 26.6):
  ```
  xcodebuild -downloadPlatform iOS -buildVersion 15.5
  xcrun simctl list runtimes
  ```
  Not executed (multi-GB). Mechanism confirmed by
  https://help.apple.com/xcode/mac/current/en.lproj/deva7379ae35.html. The
  sim is 15.5, not 15.8; 15.8.x are security-only, so API behavior matches.
- Device type: `iPad Air 2` and `iPad mini 4` are not in `simctl list
  devicetypes` on Xcode 26.6. Use `iPad Pro (9.7-inch)` (same 768x1024pt
  @2x as the mini 4) or `iPad (9th generation)` (810x1080pt).
- Swift 6.2 compiles for an iOS 15 target; language mode is independent of
  deployment target. Swift concurrency (async/await, actors, Task)
  back-deploys to iOS 13 since Xcode 13.2
  (https://www.swiftbysundell.com/special/swift-concurrency-backward-compatibility/).
  `URLSession.data(for:)` async is iOS 15+.
- Runtime-gated features that will NOT work on iOS 15: `Clock`/`Duration`/
  `Task.sleep(for:)` and `Regex` need iOS 16
  (https://github.com/swiftlang/swift-evolution/blob/main/proposals/0374-clock-sleep-for.md);
  use `Task.sleep(nanoseconds:)` and `NSRegularExpression`.
  `@Observable`/Observation and SwiftData are iOS 17.
- `#Preview` works for SwiftUI views with pre-17 targets
  (https://developer.apple.com/documentation/xcode-release-notes/xcode-15-release-notes).
  Swift Testing declares `.iOS(.v13)` minimum; UNVERIFIED that it runs in
  an iOS 15.5 sim - use XCTest.

**SwiftUI APIs missing on iOS 15 and replacements**

| iOS 16+ API | iOS 15 replacement |
|---|---|
| `NavigationStack`, `.navigationDestination` | `NavigationView` + `.navigationViewStyle(.stack)`, `NavigationLink(isActive:)` / `tag:selection:` |
| `NavigationSplitView` | `NavigationView` with 2-3 children (columns on iPad; only reliable in landscape) or `UISplitViewController` via `UIViewControllerRepresentable` (https://useyourloaf.com/blog/swiftui-split-view-configuration/) |
| `Grid`, `Table`, `Layout`, `ViewThatFits` | `LazyVGrid`/`HStack`/`VStack`, GeometryReader |
| `Charts` | none; draw with `Canvas` (iOS 15) |
| `ShareLink`, `Transferable` | `UIActivityViewController` wrapped |
| `.scrollContentBackground(.hidden)` | `UITableView.appearance().backgroundColor = .clear` (works on 15, breaks on 16) |
| `.presentationDetents` | `UISheetPresentationController` via representable (iOS 15 UIKit has detents) |
| `TextField(axis: .vertical)` | `TextEditor` (iOS 14) |
| `.scrollDisabled`, `.scrollDismissesKeyboard`, `.toolbarBackground`, `.fontWeight` on View, `ImageRenderer`, `Gauge`, `LabeledContent` | UIKit or manual equivalents |
| iOS 17: `.scrollPosition`, `.contentMargins`, `.safeAreaPadding`, two-argument `.onChange`, `ContentUnavailableView`, `.sensoryFeedback`, `@Observable` | `ScrollViewReader` (iOS 14), padding, one-argument `.onChange`, `ObservableObject` |

Sources: https://www.hackingwithswift.com/articles/250/whats-new-in-swiftui-for-ios-16,
https://developer.apple.com/documentation/swiftui/navigationview. Backports:
https://github.com/shaps80/SwiftUIBackports.

**Available on iOS 15** (use freely): `.task`, `AsyncImage`, `@FocusState`,
`.searchable`, `.refreshable`, `.swipeActions`, `.safeAreaInset`,
`.interactiveDismissDisabled`, `Canvas`, `TimelineView`,
`.confirmationDialog`, `.textSelection`, `.dynamicTypeSize`, `.onSubmit`,
`Material` backgrounds, `.foregroundStyle`, `.bordered` buttons,
`AttributedString` + inline Markdown in `Text`, `.badge`
(https://www.hackingwithswift.com/articles/235/whats-new-in-swiftui-for-ios-15).
iOS 14: `@AppStorage`, `@SceneStorage`, `ScenePhase`, `LazyVGrid`,
`.toolbar`, `.fullScreenCover`, `.keyboardShortcut`, `TextEditor`.

Recommendation: SwiftUI for chrome, forms and question pages; UIKit
(`UISplitViewController`, `WKWebView`, `PKCanvasView`, `UITextView`)
wrapped where iOS 15 SwiftUI is weak.

## 3. Rendering typeset math and long documents

- **Native `AttributedString`/`Text` is not enough**: iOS 15 Markdown is
  inline-only (emphasis, links, code); no headings, lists, tables, math
  (https://developer.apple.com/forums/thread/682711). Native LaTeX:
  `SwiftMath`/`iosMath` render TeX with CoreText (iOS 13+)
  (https://github.com/mgriebling/SwiftMath). Good for standalone equations;
  paragraph flow with inline math becomes your layout problem.
- **WKWebView + bundled KaTeX is the pragmatic reader.** KaTeX renders
  synchronously with no reflow and bundles locally (https://katex.org/);
  prefer pre-rendered HTML so the web view only needs `katex.min.css` +
  fonts (https://katex.org/docs/libs). Load with
  `loadFileURL(_:allowingReadAccessTo:)` passing the directory holding
  HTML/CSS/JS/fonts (https://www.hackingwithswift.com/articles/112/the-ultimate-guide-to-wkwebview).
  A `WKURLSchemeHandler` custom scheme (iOS 11+) avoids file-URL quirks.
- **JS bridge**: `WKUserContentController.add(_:name:)`; JS calls
  `window.webkit.messageHandlers.<name>.postMessage(...)`; the controller
  strongly retains handlers, use a weak proxy. `WKScriptMessageHandlerWithReply`
  and `callAsyncJavaScript` are iOS 14+
  (https://developer.apple.com/documentation/webkit/wkscriptmessagehandler).
  Inject a `WKUserScript` at document end for position tracking
  (IntersectionObserver over section ids) and report to Swift.
- **Content blocking**: `WKContentRuleListStore` (iOS 11+) blocks network
  loads; for local-only content also intercept `decidePolicyFor`.
- **Scrolling**: do not nest the web view in a SwiftUI ScrollView; make the
  web view the scroller. The "measure scrollHeight and resize" trick is
  fragile with reflow, Dynamic Type and long chapters. Add gesture
  recognizers to `webView.scrollView` for tap zones.
- **Pagination vs continuous scroll**: continuous is simpler and matches
  Books' Scroll mode. Paginated = CSS multi-column with `column-width` =
  viewport width plus `scrollView.isPagingEnabled = true`
  (https://ahmedk92.github.io/2017/11/03/WKWebView-Horizontal-Paging.html).
  On an A8-class device: one chapter per web view, pre-rendered, fonts
  subset, lazy images. On iOS < 16.4 Safari Web Inspector attaches to debug
  builds without `isInspectable`.

## 4. PencilKit and Scribble on iOS 15

- `PKCanvasView` is a `UIScrollView` subclass; `drawingPolicy` (iOS 14+)
  defaults to `.pencilOnly` on iPad; `.anyInput` accepts finger/capacitive
  stylus (https://developer.apple.com/documentation/pencilkit/pkcanvasview/drawingpolicy).
  With `.anyInput` there is no palm rejection; put the canvas in a
  fixed-size answer box, not over scrolling content.
- Export: `PKDrawing.image(from:scale:)` returns a UIImage (iOS 13+);
  `dataRepresentation()` for storage; `PKDrawing` is Codable
  (https://developer.apple.com/documentation/pencilkit/pkdrawing). Set
  `overrideUserInterfaceStyle = .light` before export so dark-mode ink
  inversion does not produce white-on-transparent PNGs (UNVERIFIED as the
  cleanest fix). `PKStroke.path` exposes the points, so strokes can also be
  shipped as normalized polylines - which is exactly the server's existing
  `/ink` contract.
- Tool picker: `PKToolPicker()` + `setVisible(_:forFirstResponder:)`; for an
  answer box a single pen plus clear is enough.
- Scribble: free in `UITextField`/`UITextView`/SwiftUI `TextField`.
  Apple's Scribble sample "must be run on a physical device with Apple
  Pencil"; nothing Pencil-related is testable in the simulator.

## 5. Persistence and lifecycle on iOS 15

- `UserDefaults` for small state (last book id, reading position, theme);
  `@AppStorage`/`@SceneStorage` bind it. Files in Application Support for
  JSON/Codable per-book state and cached chapters; `Data.write(options:
  .atomic)`. Core Data only for queries across many books; SwiftData is iOS
  17. Mark caches `isExcludedFromBackup`.
- Lifecycle: `@Environment(\.scenePhase)` with `active`, `inactive`,
  `background`; save on `.background` AND `.inactive` (Split View resizing
  and the app switcher hit that first)
  (https://developer.apple.com/documentation/swiftui/scenephase). Save
  reading position continuously (debounced) from the web view's scroll and
  section reports.

## 6. Automated Simulator loop (commands verified on Xcode 26.6)

```
# one-time
xcodebuild -downloadPlatform iOS -buildVersion 15.5
xcrun simctl create "chiron-ipad-15" "com.apple.CoreSimulator.SimDeviceType.iPad-Pro--9-7-inch-" \
    com.apple.CoreSimulator.SimRuntime.iOS-15-5
xcrun simctl boot "chiron-ipad-15"

# build + install + launch
xcodebuild -scheme Chiron -destination 'platform=iOS Simulator,name=chiron-ipad-15' \
    -derivedDataPath build-sim build
xcrun simctl install booted build-sim/Build/Products/Debug-iphonesimulator/Chiron.app
xcrun simctl launch --console booted dev.mjbraun.chiron selftest   # argv passed to app
xcrun simctl terminate booted dev.mjbraun.chiron

# drive and observe
xcrun simctl openurl booted 'chiron://book/data/contents'        # deep links
xcrun simctl io booted screenshot shot.png
xcrun simctl io booted recordVideo clip.mp4
xcrun simctl ui booted appearance dark|light
xcrun simctl ui booted content_size extra-large                  # or accessibility-extra-large
xcrun simctl ui booted increase_contrast enabled
xcrun simctl status_bar booted override --time 9:41
xcrun simctl get_app_container booted dev.mjbraun.chiron data     # inspect saved state
xcrun simctl spawn booted log stream --predicate 'subsystem == "com.mjbraun.chiron"'
```
Sources: local `xcrun simctl <cmd>` help; https://nshipster.com/simctl/. Env
vars reach the app via the `SIMCTL_CHILD_` prefix (used by
`scripts/sim-run.sh` today). Rotation and Split View have no simctl
command; rotation is Cmd-Left/Right in Simulator (or force orientation via a
debug flag).

**Driving the UI.** `simctl` cannot tap. Options:
- XCUITest: `xcodebuild test -only-testing:...`, accessibility identifiers,
  `XCUIScreen.main.screenshot()`; slow per run but real touches, keyboard,
  rotation.
- Meta idb (installed: `/opt/homebrew/bin/idb_companion`, `idb` via asdf):
  `idb ui describe-all` dumps the accessibility tree, `idb ui tap x y`, text
  input (https://fbidb.io/docs/idb/commands/). Best fit for an agent loop;
  MathText web views never appear in the AX tree - read those from a
  screenshot.
- In-app debug harness: DEBUG builds start an `NWListener` HTTP server on
  localhost; the simulator shares the Mac's loopback, so `curl
  localhost:PORT/state` and `/navigate?to=` work from the shell. Cheapest
  and deterministic; dumps model state screenshots cannot show. Gate behind
  a launch argument and `#if DEBUG`.
- Recommended: harness for state/navigation plus `simctl io screenshot` for
  visuals, idb for taps, XCUITest for a few gesture-level flows.

**Simulator gaps**: no Apple Pencil, no Scribble, no pressure; hardware
keyboard via I/O > Keyboard; trackpad pointer via the toolbar Capture Cursor
button on iPad sims; Split View testable by hand only. The iOS 26.5 sim is
not a substitute for the iOS 15.5 sim, and neither replaces a device pass on
the mini 4 (A8, 2 GB RAM).

## Unverified or not found

- Whether Xcode 27 will still install iOS 15/16 runtimes (index caps at
  26.99, suggesting no).
- Actual download and boot of the iOS 15.5 runtime on this Mac (not
  executed).
- Swift Testing running inside an iOS 15.5 simulator.
- Apple Books tap-zone geometry (undocumented).
