import QtQuick 2.15
import QtQuick.Controls 2.15
import net.asivery.AppLoad 1.0
import net.asivery.Framebuffer 1.0

// Chiron on reMarkable Paper Pro - reading client.
//
// The server renders the chapter into page images at device resolution;
// this frontend is a pager over them. Tap the right edge for the next
// page, the left edge for the previous one. The e-ink UI language is
// plain black-on-white with no animation.
Rectangle {
    id: root
    color: "white"

    signal close
    function unloading() {}

    property string serverBase: "http://localhost:8082"
    property string subject: "ai"
    // Baked auth token for a public server (CHIRON_TOKEN at build time).
    // Rides as a query parameter: QML Image elements cannot send headers.
    property string authToken: ""
    function tok(u) {
        if (authToken === "") return u
        return u + (u.indexOf("?") >= 0 ? "&" : "?") + "token=" + authToken
    }

    // Low-latency ink: a backend process (chiron-ink) draws pen strokes
    // into a qtfb framebuffer with the e-ink fast waveform - QML rendering
    // never gets that path. The frontend pushes zones + stored strokes per
    // page and pulls everything back before submissions. Without a backend
    // (the PC emulator) the QML canvas fallback below handles ink.
    property bool backendAlive: false
    property var pendingFlushAction: null

    AppLoad {
        id: inkBackend
        applicationID: "dev.mjbraun.chiron"
        onMessageReceived: (type, contents) => {
            if (type === 100) {
                root.backendAlive = true
                root.pushInkState()
            } else if (type === 101) {
                const r = JSON.parse(contents)
                root.setMap("inkByPage", r.page, r.strokes || [])
                if (root.pendingFlushAction !== null) {
                    const act = root.pendingFlushAction
                    root.pendingFlushAction = null
                    flushTimeout.stop()
                    act()
                }
            }
        }
    }
    Timer {
        // A backend hiccup must not hard-lock submission: proceed with
        // whatever strokes the frontend already has.
        id: flushTimeout
        interval: 2000
        onTriggered: {
            if (root.pendingFlushAction !== null) {
                const act = root.pendingFlushAction
                root.pendingFlushAction = null
                act()
            }
        }
    }

    function pushInkState() {
        if (!backendAlive) return
        const items = itemsForPage(page)
        const zones = []
        var enabled = mode === "reading" && browseUnit === "" && kbItem === ""
        if (items.length > 0) {
            for (var i = 0; i < items.length; i++) {
                const it = items[i]
                if (it.kind !== "mcq")
                    zones.push([it.rect[0], it.rect[1], it.rect[2],
                                it.rect[3] - it.strip])
            }
            if (zones.length === 0) enabled = false
        } else {
            // Prose pages: ink anywhere in the content area, never the
            // margins - the chrome (page turns, running head) lives there.
            zones.push([110, 100, 1400, 1960])
        }
        inkBackend.sendMessage(1, JSON.stringify({
            enabled: enabled, page: page, zones: zones,
            strokes: strokesForPage(page)
        }))
    }
    // Pull the backend's strokes (tagged with its current page), then act.
    function inkFlush(act) {
        if (!backendAlive) { act(); return }
        pendingFlushAction = act
        flushTimeout.restart()
        // Never an empty payload: AppLoad transmits "" as a zero-length
        // packet, which reads exactly like EOF on the backend's socket.
        inkBackend.sendMessage(2, "{}")
    }

    // Bundled faces (SPEC §0.3): Source Sans 3 for every control label,
    // Source Serif 4 for native prose. Loaded per weight - the desktop TTFs
    // register distinct family names, so each use site names its loader.
    FontLoader { id: fontSans; source: "fonts/SourceSans3-Regular.ttf" }
    FontLoader { id: fontSansMed; source: "fonts/SourceSans3-Medium.ttf" }
    FontLoader { id: fontSansSemi; source: "fonts/SourceSans3-Semibold.ttf" }
    FontLoader { id: fontSansBold; source: "fonts/SourceSans3-Bold.ttf" }
    FontLoader { id: fontSerif; source: "fonts/SourceSerif4-Regular.ttf" }
    FontLoader { id: fontSerifSemi; source: "fonts/SourceSerif4-Semibold.ttf" }
    FontLoader { id: fontSerifIt; source: "fonts/SourceSerif4-It.ttf" }

    // SPEC dimensions are page pixels at 1620x2160. ps maps them through
    // the painted page (1.0 on the tablet, smaller in the emulator window);
    // px0/py0 are the painted page's screen origin.
    property real ps: pageImage.paintedWidth > 0 ? pageImage.paintedWidth / 1620
                     : Math.min(width / 1620, height / 2160)
    property real px0: pageImage.x + (pageImage.width - pageImage.paintedWidth) / 2
    property real py0: pageImage.y + (pageImage.height - pageImage.paintedHeight) / 2
    // Fully-native screens (placement, waits, errors) have no page image;
    // they lay out on the same virtual page, centered.
    property real nx0: (width - 1620 * ps) / 2
    property real ny0: (height - 2160 * ps) / 2
    // Border widths never scale below one device pixel.
    function bw(w) { return Math.max(1, Math.round(w * ps)) }

    // Native control anatomy (SPEC §0.7): flat, square, black-on-white.
    // "Pressed" is a momentary inversion - selected XOR pressed fills black.
    component ConfPill: Rectangle {
        id: pill
        property string label
        property bool selected: false
        property bool muted: false
        signal tapped()
        height: 64 * root.ps
        width: pillText.implicitWidth + 56 * root.ps
        color: pill.selected !== pillArea.pressed ? "#000000" : "#FFFFFF"
        border.width: pill.muted && !pill.selected ? root.bw(1) : root.bw(2)
        border.color: pill.muted && !pill.selected ? "#777777" : "#000000"
        Text {
            id: pillText
            anchors.centerIn: parent
            text: pill.label
            font.family: fontSansSemi.name
            font.pixelSize: 26 * root.ps
            color: pill.selected !== pillArea.pressed ? "#FFFFFF"
                 : (pill.muted ? "#777777" : "#000000")
        }
        MouseArea { id: pillArea; anchors.fill: parent; onClicked: pill.tapped() }
    }

    component IdkButton: Rectangle {
        id: idkBtn
        property bool selected: false
        signal tapped()
        height: 64 * root.ps
        width: idkText.implicitWidth + 56 * root.ps
        color: idkBtn.selected !== idkArea.pressed ? "#000000" : "#FFFFFF"
        border.width: idkBtn.selected ? root.bw(2) : root.bw(1)
        border.color: idkBtn.selected ? "#000000" : "#666666"
        Text {
            id: idkText
            anchors.centerIn: parent
            text: "I don't know"
            font.family: fontSansMed.name
            font.pixelSize: 26 * root.ps
            color: idkBtn.selected !== idkArea.pressed ? "#FFFFFF" : "#444444"
        }
        MouseArea { id: idkArea; anchors.fill: parent; onClicked: idkBtn.tapped() }
    }

    component LetterButton: Rectangle {
        id: lb
        property string letter
        property bool selected: false
        property bool muted: false
        signal tapped()
        width: 64 * root.ps
        height: 64 * root.ps
        color: lb.selected !== lbArea.pressed ? "#000000" : "#FFFFFF"
        border.width: lb.muted && !lb.selected ? root.bw(1) : root.bw(2)
        border.color: lb.muted && !lb.selected ? "#777777" : "#000000"
        Text {
            anchors.centerIn: parent
            text: lb.letter
            font.family: fontSansBold.name
            font.pixelSize: 30 * root.ps
            color: lb.selected !== lbArea.pressed ? "#FFFFFF"
                 : (lb.muted ? "#777777" : "#000000")
        }
        MouseArea { id: lbArea; anchors.fill: parent; onClicked: lb.tapped() }
    }

    component ActionButton: Rectangle {
        id: ab
        property string label
        property bool active: true
        signal tapped()
        height: 64 * root.ps
        width: abText.implicitWidth + 72 * root.ps
        color: ab.active && abArea.pressed ? "#000000" : "#FFFFFF"
        border.width: ab.active ? root.bw(2) : root.bw(1)
        border.color: ab.active ? "#000000" : "#777777"
        Text {
            id: abText
            anchors.centerIn: parent
            text: ab.label
            font.family: fontSansSemi.name
            font.pixelSize: 28 * root.ps
            color: ab.active ? (abArea.pressed ? "#FFFFFF" : "#000000") : "#777777"
        }
        MouseArea {
            id: abArea
            anchors.fill: parent
            onClicked: if (ab.active) ab.tapped()
        }
    }

    // The quiet action variant (SPEC §6): the path the design does not push.
    component QuietButton: Rectangle {
        id: qb
        property string label
        signal tapped()
        height: 64 * root.ps
        width: qbText.implicitWidth + 72 * root.ps
        color: qbArea.pressed ? "#000000" : "#FFFFFF"
        border.width: root.bw(1)
        border.color: "#666666"
        Text {
            id: qbText
            anchors.centerIn: parent
            text: qb.label
            font.family: fontSansMed.name
            font.pixelSize: 28 * root.ps
            color: qbArea.pressed ? "#FFFFFF" : "#444444"
        }
        MouseArea { id: qbArea; anchors.fill: parent; onClicked: qb.tapped() }
    }

    // Placement rating row (SPEC §0.7): 1400x112, a 96px number cell with
    // a full-height divider, label to its right. Exactly one selectable;
    // a second tap on the selected row does not deselect.
    component RatingRow: Rectangle {
        id: rr
        property int number
        property string label
        property bool selected: false
        signal tapped()
        width: 1400 * root.ps
        height: 112 * root.ps
        color: rr.selected !== rrArea.pressed ? "#000000" : "#FFFFFF"
        border.width: rr.selected ? root.bw(2) : root.bw(1)
        border.color: rr.selected ? "#000000" : "#666666"
        Text {
            x: 0
            width: 96 * root.ps
            height: parent.height
            horizontalAlignment: Text.AlignHCenter
            verticalAlignment: Text.AlignVCenter
            text: rr.number
            font.family: fontSansBold.name
            font.pixelSize: 30 * root.ps
            color: rr.selected !== rrArea.pressed ? "#FFFFFF" : "#000000"
        }
        Rectangle {
            x: 96 * root.ps
            width: root.bw(1)
            height: parent.height
            color: rr.selected ? "#555555" : "#666666"
        }
        Text {
            x: (96 + 28) * root.ps
            width: parent.width - x - 28 * root.ps
            height: parent.height
            verticalAlignment: Text.AlignVCenter
            elide: Text.ElideRight
            text: rr.label
            font.family: fontSansSemi.name
            font.pixelSize: 30 * root.ps
            color: rr.selected !== rrArea.pressed ? "#FFFFFF" : "#000000"
        }
        MouseArea { id: rrArea; anchors.fill: parent; onClicked: rr.tapped() }
    }

    component PageTurn: Rectangle {
        id: pt
        property string glyph
        property bool active: true
        signal tapped()
        width: 64 * root.ps
        height: 64 * root.ps
        color: pt.active && ptArea.pressed ? "#000000" : "#FFFFFF"
        border.width: root.bw(1)
        border.color: pt.active ? "#666666" : "#777777"
        Text {
            anchors.centerIn: parent
            text: pt.glyph
            font.family: fontSans.name
            font.pixelSize: 34 * root.ps
            color: pt.active ? (ptArea.pressed ? "#FFFFFF" : "#333333") : "#777777"
        }
        MouseArea {
            id: ptArea
            anchors.fill: parent
            onClicked: if (pt.active) pt.tapped()
        }
    }

    // "loading" | "reading" | "submitting" | "results" | "error"
    property string mode: "loading"
    property string errorText: ""
    property string subjectTitle: ""
    property string chapterTitle: ""
    property string chapterUnit: ""
    property string pagesHash: ""
    property int pageCount: 0
    property int page: 0
    // Item -> page map from the server; ink on an item's page belongs to it.
    property var itemPages: []
    property bool bootstrapTried: false
    // The interactive page geometry contract from the server: every answer
    // box reserves a bottom strip for these controls.
    property var layoutC: null
    // Non-null when the chapter is the placement screener: rendered natively,
    // the question box is the UI.
    property var screener: null
    property var checkinResult: null
    // Typeset results pages (server-rendered): {count, hash, action}.
    property var resultsMeta: null
    property int resultsPage: 0
    // Contents / spine (SPEC §7): server-rendered rows, native tap targets.
    property var contentsMeta: null
    // Non-empty while reading a past chapter read-only from the contents.
    property string browseUnit: ""
    // The learner's actual current unit and reading position, restored when
    // contents or a browsed chapter is left.
    property string homeUnit: ""
    property int savedPage: 0
    // Reading position to restore after the next loadMeta, honored only if
    // the chapter comes back unchanged (same unit and page hash).
    property int pendingRestorePage: -1
    // One reload per stale-page 404, so a persistent failure cannot loop.
    property bool pageErrorReloading: false
    // Pacing (SPEC §6): the server suggests breaks from reported reading
    // time; actual break minutes ride the next check-in.
    property var breakSuggestion: null
    property double chapterStartMs: 0
    property double breakStartMs: 0
    property double pendingBreakMinutes: 0
    // Answer state keyed by item id - pages hold several items now.
    property var idkByItem: ({})
    property var selByItem: ({})
    property var confByItem: ({})
    // Typed answers (the on-screen keyboard alternative to ink). Every
    // entry field offers both; a typed answer goes to grading verbatim.
    property var textByItem: ({})
    // Item id whose keyboard is open; the page shifts up so the active
    // field stays visible above the keyboard.
    property string kbItem: ""
    property bool kbShift: false
    property real viewShift: 0

    function openKb(id) {
        const it = itemById(id)
        if (!it || it.kind === "mcq") return
        kbItem = id
        // Shift the page so the item's box bottom clears the keyboard.
        const pyBase = (height - pageImage.paintedHeight) / 2
        const boxBottom = pyBase + (it.rect[1] + it.rect[3]) * ps
        const kbTop = height - osk.height
        viewShift = Math.min(0, kbTop - 24 * ps - boxBottom)
    }
    function closeKb() {
        kbItem = ""
        kbShift = false
        viewShift = 0
    }
    function oskPress(k) {
        if (kbItem === "") return
        var cur = textByItem[kbItem] || ""
        if (k === "⌫") cur = cur.slice(0, -1)
        else if (k === "⇧") { kbShift = !kbShift; return }
        else if (k === "space") cur += " "
        else if (k === "done") { closeKb(); return }
        else {
            cur += kbShift ? k.toUpperCase() : k
            kbShift = false
        }
        setMap("textByItem", kbItem, cur)
        // Typing is an attempt: it clears IDK, like tapping a letter does.
        if (cur !== "") setMap("idkByItem", kbItem, false)
    }

    // Copy-on-write: assigning the SAME object reference back to a var
    // property does not signal a change, so bindings (button checked state,
    // Check in enabled) never re-evaluate. A fresh object every write does.
    function setMap(name, key, v) {
        var old = root[name]
        var m = {}
        for (var k in old) m[k] = old[k]
        m[key] = v
        root[name] = m
    }

    function itemsForPage(p) {
        var out = []
        for (var i = 0; i < itemPages.length; i++)
            if (itemPages[i].page === p) out.push(itemPages[i])
        return out
    }

    function itemById(id) {
        for (var i = 0; i < itemPages.length; i++)
            if (itemPages[i].item === id) return itemPages[i]
        return null
    }

    // IDK and the four confidence levels are one 5-way exclusive choice
    // (SPEC §2.3); selecting IDK on an MCQ also clears the letter. Muted
    // controls stay tappable - tapping restores attempt mode.
    function tapIdk(id) {
        setMap("idkByItem", id, true)
        setMap("confByItem", id, undefined)
        const it = itemById(id)
        if (it && it.kind === "mcq") setMap("selByItem", id, undefined)
    }
    function tapConf(id, v) {
        setMap("confByItem", id, v)
        setMap("idkByItem", id, false)
    }
    function tapLetter(id, i) {
        setMap("selByItem", id, i)
        setMap("idkByItem", id, false)
    }

    // Answered: IDK, or a confidence level (constructed - the transcriber
    // handles blank ink), or letter plus confidence (MCQ).
    function answered(it) {
        if (idkByItem[it.item] === true) return true
        if (!confByItem[it.item]) return false
        if (it.kind === "mcq")
            return selByItem[it.item] !== undefined && selByItem[it.item] !== null
        return true
    }
    function unansweredCount() {
        var n = 0
        for (var i = 0; i < itemPages.length; i++)
            if (!answered(itemPages[i])) n++
        return n
    }

    Timer {
        id: retryMeta
        interval: 3000
        onTriggered: root.loadMeta()
    }

    function loadMeta() {
        mode = "loading"
        const browsing = browseUnit !== ""
        const xhr = new XMLHttpRequest()
        xhr.onreadystatechange = function() {
            if (xhr.readyState !== XMLHttpRequest.DONE) return
            if (browsing && xhr.status !== 200) {
                // A past chapter that cannot load is a dead end, not an
                // error state: fall back to the contents.
                browseUnit = ""
                openContents()
                return
            }
            if (xhr.status === 200) {
                const m = JSON.parse(xhr.responseText)
                if (m.authoring) {
                    // Grades came back instantly; the chapter is still being
                    // written. Keep the learner informed and poll.
                    if (!authoring) {
                        authoring = true
                        authoringLong = false
                        authoringLongTimer.restart()
                    }
                    retryMeta.restart()
                    return
                }
                authoring = false
                authoringLong = false
                authoringLongTimer.stop()
                if (m.authoring_error) {
                    errorText = "Chapter authoring failed:\n" + m.authoring_error
                    mode = "error"
                    return
                }
                // Restore the reading position only when the chapter truly
                // did not change underneath the reader.
                const unchanged = m.unit === chapterUnit && m.hash === pagesHash
                chapterTitle = m.title
                subjectTitle = m.subject_title || ""
                chapterUnit = m.unit
                pageCount = m.count
                pagesHash = m.hash
                itemPages = m.items || []
                screener = m.screener || null
                layoutC = m.layout || null
                page = unchanged && pendingRestorePage >= 0 ? pendingRestorePage : 0
                pendingRestorePage = -1
                pageErrorReloading = false
                if (!browsing) {
                    homeUnit = m.unit
                    chapterStartMs = Date.now()
                }
                mode = screener && !browsing ? "screener" : "reading"
                console.log("[chiron] chapter:", m.unit, m.count, "pages,",
                            itemPages.length, "answer pages")
            } else if (xhr.status === 404 && !root.bootstrapTried) {
                // Fresh learner: no chapter exists yet. Start the book and
                // retry once - the start exchange delivers the first chapter
                // without any model call.
                root.bootstrapTried = true
                const boot = new XMLHttpRequest()
                boot.onreadystatechange = function() {
                    if (boot.readyState !== XMLHttpRequest.DONE) return
                    if (boot.status === 200) {
                        loadMeta()
                    } else {
                        errorText = "Could not start the book (" + boot.status + ")"
                        mode = "error"
                    }
                }
                boot.open("POST", tok(serverBase + "/exchange"))
                boot.setRequestHeader("Content-Type", "application/json")
                boot.send(JSON.stringify({ subject: subject, phase: "start" }))
            } else {
                errorText = "No chapter available (" + xhr.status + ").\n"
                          + "Server: " + serverBase
                mode = "error"
                console.log("[chiron] meta failed:", xhr.status)
            }
        }
        xhr.open("GET", tok(serverBase + "/pages/" + subject
                 + (browsing ? "?unit=" + browseUnit : "")))
        xhr.send()
    }

    // Contents / spine: fetch the rendered page and the row map, remembering
    // where reading left off.
    function openContents() {
        if (mode === "reading" && browseUnit === "") savedPage = page
        const xhr = new XMLHttpRequest()
        xhr.onreadystatechange = function() {
            if (xhr.readyState !== XMLHttpRequest.DONE) return
            if (xhr.status !== 200) return
            contentsMeta = JSON.parse(xhr.responseText)
            mode = "contents"
        }
        xhr.open("GET", tok(serverBase + "/pages/" + subject + "/contents"))
        xhr.send()
    }

    // A tap on a written contents row opens that chapter: the current one
    // resumes where reading left off, cleared ones open read-only. The home
    // chapter ALWAYS goes through loadMeta - after a check-in the delivered
    // chapter has changed, and resuming a cached meta once composited a
    // previous check's answer strips over the new chapter's pages. The
    // reading position survives when the chapter really is unchanged.
    function contentsGoto(row) {
        if (!row || row.state === "unwritten") return
        if (row.unit === homeUnit) {
            browseUnit = ""
            pendingRestorePage = savedPage
            loadMeta()
            return
        }
        browseUnit = row.unit
        loadMeta()
    }

    Component.onCompleted: {
        loadMeta()
        const en = new XMLHttpRequest()
        en.onreadystatechange = function() {
            if (en.readyState !== XMLHttpRequest.DONE || !en.responseText) return
            const val = en.responseText.trim()
            // The enable file names the drive server, which also becomes the
            // book server for the session - the harness owns both.
            if (val.indexOf("http") === 0) {
                serverBase = val
                bootstrapTried = false
                loadMeta()
            }
            drive.running = true
        }
        en.open("GET", "file:///tmp/chiron-drive/enable")
        en.send()
    }

    // Ink, kept per page in page-image coordinates (0..1 normalized), so
    // strokes survive window resizes and can later be shipped to the server
    // against the answer-region geometry.
    property var inkByPage: ({})

    function strokesForPage(p) {
        if (!inkByPage[p]) inkByPage[p] = []
        return inkByPage[p]
    }

    // Reading view
    Image {
        id: pageImage
        // Not anchored: y carries the keyboard view-shift, and every
        // overlay tracks it through px0/py0.
        x: 0
        y: root.viewShift
        width: parent.width
        height: parent.height
        visible: root.mode === "reading"
        fillMode: Image.PreserveAspectFit
        // The hash pins the URL to the chapter revision, so stale cached
        // pages can never show for a regenerated chapter.
        source: root.mode === "reading"
            ? root.tok(root.serverBase + "/pages/" + root.subject + "/" + root.page
              + "?v=" + root.pagesHash
              + (root.browseUnit !== "" ? "&unit=" + root.browseUnit : ""))
            : ""
        asynchronous: true
        cache: true
        // The server refuses page requests carrying a stale chapter hash
        // (404). That means this client's meta is out of date: refresh it
        // instead of showing a blank page under live answer strips.
        onStatusChanged: {
            if (status === Image.Error && root.mode === "reading"
                && !root.pageErrorReloading) {
                root.pageErrorReloading = true
                root.loadMeta()
            }
        }
    }

    Canvas {
        id: ink
        anchors.fill: pageImage
        visible: root.mode === "reading" && !root.backendAlive

        // Pen latency lives or dies on damage size: painting synchronously
        // and marking ONLY the new segment's rect dirty keeps each update
        // tiny, so the e-ink compositor can take its fast refresh path. A
        // full clear-and-redraw per point (the naive approach) produces
        // whole-canvas damage and the slow waveform on every pen movement.
        renderStrategy: Canvas.Immediate
        renderTarget: Canvas.Image

        property var liveStroke: null
        // How many liveStroke points are already on the canvas.
        property int painted: 0
        property bool fullRepaint: true

        function pen(ctx) {
            ctx.strokeStyle = "black"
            ctx.lineWidth = 3
            ctx.lineCap = "round"
            ctx.lineJoin = "round"
        }

        function requestFull() {
            fullRepaint = true
            requestPaint()
        }

        // Mark only the last segment's bounding box dirty and paint it.
        function extendLive() {
            if (!liveStroke || liveStroke.length === 0) return
            const a = liveStroke[Math.max(0, liveStroke.length - 2)]
            const b = liveStroke[liveStroke.length - 1]
            const x0 = Math.min(a.x, b.x) * width - 6
            const y0 = Math.min(a.y, b.y) * height - 6
            const w = Math.abs(a.x - b.x) * width + 12
            const h = Math.abs(a.y - b.y) * height + 12
            markDirty(Qt.rect(x0, y0, w, h))
        }

        onPaint: {
            const ctx = getContext("2d")
            if (!fullRepaint && liveStroke && painted < liveStroke.length) {
                // Incremental: draw just the new tail of the live stroke.
                pen(ctx)
                ctx.beginPath()
                const from = Math.max(0, painted - 1)
                ctx.moveTo(liveStroke[from].x * width, liveStroke[from].y * height)
                for (var i = from + 1; i < liveStroke.length; i++)
                    ctx.lineTo(liveStroke[i].x * width, liveStroke[i].y * height)
                ctx.stroke()
                painted = liveStroke.length
                return
            }
            ctx.clearRect(0, 0, width, height)
            pen(ctx)
            const all = root.strokesForPage(root.page)
            const drawStroke = function(s) {
                if (s.length < 2) return
                ctx.beginPath()
                ctx.moveTo(s[0].x * width, s[0].y * height)
                for (var i = 1; i < s.length; i++)
                    ctx.lineTo(s[i].x * width, s[i].y * height)
                ctx.stroke()
            }
            for (var j = 0; j < all.length; j++) drawStroke(all[j])
            if (liveStroke) drawStroke(liveStroke)
            painted = liveStroke ? liveStroke.length : 0
            fullRepaint = false
        }

        MouseArea {
            anchors.fill: parent
            function endStroke() {
                if (ink.liveStroke && ink.liveStroke.length > 1)
                    root.strokesForPage(root.page).push(ink.liveStroke)
                ink.liveStroke = null
                ink.painted = 0
                // The stroke is already on the canvas; nothing to repaint.
            }
            onPressed: (e) => {
                if (!root.inkAllowedAt(e.x / ink.width, e.y / ink.height)) return
                ink.liveStroke = [{x: e.x / ink.width, y: e.y / ink.height}]
                ink.painted = 0
                ink.extendLive()
            }
            onPositionChanged: (e) => {
                if (!ink.liveStroke) return
                // Leaving the ink zone ends the stroke - nothing is ever
                // drawn over a control strip (SPEC §2.3).
                if (!root.inkAllowedAt(e.x / ink.width, e.y / ink.height)) {
                    endStroke()
                    return
                }
                ink.liveStroke.push({x: e.x / ink.width, y: e.y / ink.height})
                ink.extendLive()
            }
            onReleased: endStroke()
        }
    }

    onPageChanged: {
        ink.requestFull()
        // Persist the old page's backend strokes, then arm the new page.
        if (backendAlive) { inkBackend.sendMessage(2, "{}"); pushInkState() }
    }
    onModeChanged: {
        if (mode === "reading") ink.requestFull()
        pushInkState()
    }
    onKbItemChanged: pushInkState()

    // The backend's ink overlay: a transparent RGBA framebuffer the
    // chiron-ink process draws into with fast-waveform refreshes. Sits over
    // the painted page; all tappable chrome is declared later, so stays
    // above it.
    FBController {
        // Full-screen at native resolution with scaling off: the scaled
        // path forces smooth-transform drawImage on every dirty rect
        // (research brief); the unscaled path is a plain blit. Only ever
        // visible on the device, where the app IS 1620x2160.
        visible: root.mode === "reading" && root.backendAlive
        framebufferID: root.backendAlive ? 0x43484952 : -1
        x: 0
        y: 0
        width: 1620
        height: 2160
        allowScaling: false
    }

    // Bottom chrome band (SPEC §1): page turns at x 110/186, action button
    // right-aligned to x 1510, all 64px controls centered in the bottom
    // margin (y 2078). Page-side folio owns the center.
    PageTurn {
        visible: root.mode === "reading" && root.pageCount > 1
        glyph: "‹"
        active: root.page > 0
        x: root.px0 + 110 * root.ps
        y: root.py0 + 2078 * root.ps
        onTapped: root.page--
    }
    PageTurn {
        visible: root.mode === "reading" && root.pageCount > 1
        glyph: "›"
        active: root.page < root.pageCount - 1
        x: root.px0 + 186 * root.ps
        y: root.py0 + 2078 * root.ps
        onTapped: root.page++
    }
    ActionButton {
        id: checkInBtn
        visible: root.mode === "reading" && root.browseUnit === ""
                 && root.page === root.pageCount - 1
                 && root.itemPages.length > 0
        label: "Check in"
        active: root.unansweredCount() === 0
        x: root.px0 + 1510 * root.ps - width
        y: root.py0 + 2078 * root.ps
        onTapped: root.checkIn()
    }
    Text {
        visible: checkInBtn.visible && !checkInBtn.active
        text: root.unansweredCount() + (root.unansweredCount() === 1
              ? " item still needs an answer" : " items still need an answer")
        font.family: fontSans.name
        font.pixelSize: 22 * root.ps
        color: "#777777"
        anchors.verticalCenter: checkInBtn.verticalCenter
        x: checkInBtn.x - width - 16 * root.ps
    }

    // Placement (SPEC §4): fully native. Running head, serif intro,
    // question, five rating rows, footnote; Check in stays disabled until
    // a level is chosen.
    Item {
        visible: root.mode === "screener"
        anchors.fill: parent

        Text {
            x: root.nx0 + 110 * root.ps
            y: root.ny0 + 40 * root.ps
            text: root.subjectTitle.toUpperCase()
            font.family: fontSerifSemi.name
            font.pixelSize: 24 * root.ps
            font.letterSpacing: 24 * 0.08 * root.ps
            color: "#444444"
        }
        Text {
            x: root.nx0 + 1510 * root.ps - width
            y: root.ny0 + 40 * root.ps
            text: "PLACEMENT"
            font.family: fontSerifSemi.name
            font.pixelSize: 24 * root.ps
            font.letterSpacing: 24 * 0.08 * root.ps
            color: "#444444"
        }

        Column {
            x: root.nx0 + 110 * root.ps
            y: root.ny0 + 200 * root.ps
            Text {
                width: 1220 * root.ps
                text: root.screener ? root.screener.intro : ""
                font.family: fontSerif.name
                font.pixelSize: 34 * root.ps
                lineHeightMode: Text.FixedHeight
                lineHeight: 53 * root.ps
                wrapMode: Text.Wrap
                color: "#000000"
            }
            Item { width: 1; height: 72 * root.ps }
            Text {
                width: 1300 * root.ps
                text: root.screener ? root.screener.prompt : ""
                font.family: fontSerifSemi.name
                font.pixelSize: 40 * root.ps
                lineHeightMode: Text.FixedHeight
                lineHeight: 56 * root.ps
                wrapMode: Text.Wrap
                color: "#000000"
            }
            Item { width: 1; height: 52 * root.ps }
            Column {
                spacing: 20 * root.ps
                Repeater {
                    model: root.screener ? root.screener.options : []
                    delegate: RatingRow {
                        number: index + 1
                        label: modelData
                        selected: root.screener
                                  && root.selByItem[root.screener.item_id] === index
                        onTapped: root.setMap("selByItem", root.screener.item_id, index)
                    }
                }
            }
            Item { width: 1; height: 44 * root.ps }
            Text {
                width: 1220 * root.ps
                text: "This sets where the questions begin — nothing more."
                font.family: fontSerifIt.name
            font.italic: true
                font.pixelSize: 26 * root.ps
                lineHeightMode: Text.FixedHeight
                lineHeight: 36 * root.ps
                wrapMode: Text.Wrap
                color: "#666666"
            }
        }

        ActionButton {
            label: "Check in"
            active: root.screener !== null
                    && root.selByItem[root.screener.item_id] !== undefined
            x: root.nx0 + 1510 * root.ps - width
            y: root.ny0 + 2078 * root.ps
            onTapped: root.checkIn()
        }
        QuietButton {
            label: "Close"
            x: root.nx0 + 110 * root.ps
            y: root.ny0 + 2078 * root.ps
            onTapped: root.close()
        }
    }

    // Pen input is captured only inside a constructed item's box, above the
    // shelf rule; MCQ boxes take no ink. Pages without published items
    // (prose, beat boxes) are open ink room.
    function inkAllowedAt(nx, ny) {
        if (browseUnit !== "") return false // past chapters are read-only
        if (kbItem !== "") return false     // keyboard mode owns the field
        if (mode !== "reading" || layoutC === null) return true
        const items = itemsForPage(page)
        if (items.length === 0) return true
        for (var i = 0; i < items.length; i++) {
            const it = items[i]
            if (it.kind === "mcq") continue
            if (nx < it.rect[0] / layoutC.page_w) continue
            if (nx > (it.rect[0] + it.rect[2]) / layoutC.page_w) continue
            if (ny < it.rect[1] / layoutC.page_h) continue
            if (ny > (it.rect[1] + it.rect[3] - it.strip) / layoutC.page_h) continue
            return true
        }
        return false
    }

    // Strokes on a page belong to the item whose ink zone (rect minus the
    // control strip) contains their first point, renormalized to that zone
    // so the transcriber gets a tight crop instead of a mostly empty page.
    function strokesForItem(it) {
        const page = strokesForPage(it.page)
        const top = it.rect[1] / layoutC.page_h
        const zoneH = (it.rect[3] - it.strip) / layoutC.page_h
        const out = []
        for (var i = 0; i < page.length; i++) {
            const s = page[i]
            if (s.length === 0) continue
            if (s[0].y < top || s[0].y > top + zoneH) continue
            const rs = []
            for (var j = 0; j < s.length; j++)
                rs.push({ x: s[j].x, y: (s[j].y - top) / zoneH })
            out.push(rs)
        }
        return out
    }

    function checkIn() {
        // Only answerable states may submit: a check-in during loading or
        // error would ship stale item pages from a previous chapter, and
        // browsed chapters are read-only.
        if (mode !== "reading" && mode !== "screener") return
        if (browseUnit !== "") return
        if (backendAlive && mode === "reading") {
            // Pull the pen strokes the backend holds before building the
            // submission.
            inkFlush(function() { root.doCheckIn() })
            return
        }
        doCheckIn()
    }
    function doCheckIn() {
        mode = "submitting"
        const items = []
        if (screener) {
            items.push({ item_id: screener.item_id,
                         selected_index: selByItem[screener.item_id],
                         strokes: [] })
        } else
        for (var i = 0; i < itemPages.length; i++) {
            const it = itemPages[i]
            const entry = { item_id: it.item,
                            idk: idkByItem[it.item] === true,
                            aspect: it.rect[3] > it.strip
                                    ? layoutC.page_w / (it.rect[3] - it.strip) : 0.75,
                            strokes: strokesForItem(it) }
            if (selByItem[it.item] !== undefined && selByItem[it.item] !== null)
                entry.selected_index = selByItem[it.item]
            if (confByItem[it.item])
                entry.confidence = confByItem[it.item]
            const typed = (textByItem[it.item] || "").trim()
            if (typed !== "")
                entry.text = typed
            items.push(entry)
        }
        const xhr = new XMLHttpRequest()
        xhr.onreadystatechange = function() {
            if (xhr.readyState !== XMLHttpRequest.DONE) return
            if (xhr.status === 200) {
                checkinResult = JSON.parse(xhr.responseText)
                if (!checkinResult.gate) {
                    // Nothing was gated (the screener step): go straight to
                    // the freshly delivered pages.
                    inkByPage = ({}); idkByItem = ({}); selByItem = ({}); confByItem = ({})
                    textByItem = ({})
                    checkinResult = null
                    loadMeta()
                } else {
                    resultsMeta = checkinResult.results_pages || null
                    resultsPage = 0
                    breakSuggestion = checkinResult.break_suggestion || null
                    mode = "results"
                }
            } else {
                errorText = "Check-in failed (" + xhr.status + "): "
                          + xhr.responseText.substring(0, 200)
                mode = "error"
            }
        }
        xhr.open("POST", tok(serverBase + "/ink/" + subject))
        xhr.setRequestHeader("Content-Type", "application/json")
        const payload = { unit: chapterUnit, items: items,
                          chunk_minutes: chapterStartMs > 0
                              ? (Date.now() - chapterStartMs) / 60000 : 0 }
        if (pendingBreakMinutes > 0) {
            payload.break_minutes = pendingBreakMinutes
            pendingBreakMinutes = 0
        }
        xhr.send(JSON.stringify(payload))
    }

    // Tapping the running head (left slot - the chapter title) opens the
    // contents, book-style. Deliberately not the full top edge: the center
    // strip belongs to AppLoad's close-the-app drag gesture.
    MouseArea {
        visible: root.mode === "reading"
        x: root.px0
        y: root.py0
        width: 620 * root.ps
        height: 100 * root.ps
        onClicked: root.openContents()
    }

    // Per-item controls mapped into each answer box's reserved strip
    // (SPEC §2.3; regions come from the pages meta - the server enforces
    // the same geometry it publishes). IDK is ALWAYS the leftmost control
    // so the exit never moves between questions. Constructed: one centered
    // row, IDK+Type left, confidence right. MCQ: IDK then letters in row 1,
    // confidence row 2 right-aligned. Side insets 30 from the box's inner
    // edges.
    Repeater {
        model: root.mode === "reading" && root.layoutC !== null
               ? root.itemsForPage(root.page) : []
        delegate: Item {
            property var it: modelData
            property bool ik: root.idkByItem[it.item] === true
            property bool mcq: it.kind === "mcq"
            x: root.px0 + (it.rect[0] + 2) * root.ps
            y: root.py0 + (it.rect[1] + it.rect[3] - 2 - it.strip) * root.ps
            width: (it.rect[2] - 4) * root.ps
            height: it.strip * root.ps

            IdkButton {
                id: idkBtnStrip
                selected: ik
                x: 30 * root.ps
                y: mcq ? 26 * root.ps : (parent.height - height) / 2
                onTapped: root.tapIdk(it.item)
            }
            QuietButton {
                // Every entry field offers keyboard entry as well as pen.
                visible: !mcq
                label: root.kbItem === it.item ? "Done" : "Type"
                x: idkBtnStrip.x + idkBtnStrip.width + 12 * root.ps
                y: (parent.height - height) / 2
                onTapped: root.kbItem === it.item ? root.closeKb() : root.openKb(it.item)
            }
            Row {
                visible: mcq
                spacing: 12 * root.ps
                x: idkBtnStrip.x + idkBtnStrip.width + 24 * root.ps
                y: 26 * root.ps
                Repeater {
                    model: mcq ? it.options : 0
                    delegate: LetterButton {
                        letter: String.fromCharCode(65 + index)
                        selected: root.selByItem[it.item] === index
                        muted: ik
                        onTapped: root.tapLetter(it.item, index)
                    }
                }
            }
            Row {
                spacing: 12 * root.ps
                x: parent.width - width - 30 * root.ps
                y: mcq ? 106 * root.ps : (parent.height - height) / 2
                Repeater {
                    model: ["unsure", "shaky", "confident", "sure"]
                    delegate: ConfPill {
                        label: modelData
                        selected: root.confByItem[it.item] === index + 1
                        muted: ik
                        onTapped: root.tapConf(it.item, index + 1)
                    }
                }
            }
        }
    }

    // Typed answers, displayed in the item's ink zone (lower half, above
    // the shelf rule) with a cursor while the keyboard is open.
    Repeater {
        model: root.mode === "reading" && root.layoutC !== null
               ? root.itemsForPage(root.page) : []
        delegate: Text {
            property var it: modelData
            visible: it.kind !== "mcq"
                     && ((root.textByItem[it.item] || "") !== ""
                         || root.kbItem === it.item)
            x: root.px0 + (it.rect[0] + 58) * root.ps
            y: root.py0 + (it.rect[1] + (it.rect[3] - it.strip) * 0.45) * root.ps
            width: (it.rect[2] - 116) * root.ps
            height: (it.rect[3] - it.strip) * 0.5 * root.ps
            text: (root.textByItem[it.item] || "")
                  + (root.kbItem === it.item ? "▎" : "")
            wrapMode: Text.Wrap
            font.family: fontSerif.name
            font.pixelSize: 30 * root.ps
            lineHeightMode: Text.FixedHeight
            lineHeight: 42 * root.ps
            color: "#000000"
        }
    }

    // The on-screen keyboard (SPEC extension): flat monochrome keys, fixed
    // to the window bottom; the page view shifts to keep the field visible.
    Rectangle {
        id: osk
        visible: root.kbItem !== ""
        width: parent.width
        height: oskCol.height + 32 * root.ps
        y: parent.height - height
        color: "#FFFFFF"
        border.width: root.bw(1)
        border.color: "#666666"

        Column {
            id: oskCol
            spacing: 8 * root.ps
            anchors.horizontalCenter: parent.horizontalCenter
            y: 16 * root.ps

            Repeater {
                model: [
                    ["1","2","3","4","5","6","7","8","9","0"],
                    ["q","w","e","r","t","y","u","i","o","p"],
                    ["a","s","d","f","g","h","j","k","l"],
                    ["⇧","z","x","c","v","b","n","m","⌫"],
                    ["-","+","=","/","*","^","(",")",",","."]
                ]
                delegate: Row {
                    spacing: 8 * root.ps
                    anchors.horizontalCenter: parent.horizontalCenter
                    property var keys: modelData
                    Repeater {
                        model: parent.keys
                        delegate: Rectangle {
                            width: 140 * root.ps
                            height: 88 * root.ps
                            color: keyArea.pressed
                                   || (modelData === "⇧" && root.kbShift)
                                   ? "#000000" : "#FFFFFF"
                            border.width: root.bw(1)
                            border.color: "#666666"
                            Text {
                                anchors.centerIn: parent
                                text: root.kbShift && modelData.length === 1
                                      && modelData >= "a" && modelData <= "z"
                                      ? modelData.toUpperCase() : modelData
                                font.family: fontSans.name
                                font.pixelSize: 34 * root.ps
                                color: keyArea.pressed
                                       || (modelData === "⇧" && root.kbShift)
                                       ? "#FFFFFF" : "#000000"
                            }
                            MouseArea {
                                id: keyArea
                                anchors.fill: parent
                                onClicked: root.oskPress(modelData)
                            }
                        }
                    }
                }
            }
            Row {
                spacing: 8 * root.ps
                anchors.horizontalCenter: parent.horizontalCenter
                Rectangle {
                    width: 888 * root.ps
                    height: 88 * root.ps
                    color: spaceArea.pressed ? "#000000" : "#FFFFFF"
                    border.width: root.bw(1)
                    border.color: "#666666"
                    MouseArea { id: spaceArea; anchors.fill: parent; onClicked: root.oskPress("space") }
                }
                Rectangle {
                    width: 288 * root.ps
                    height: 88 * root.ps
                    color: doneArea.pressed ? "#FFFFFF" : "#000000"
                    border.width: root.bw(2)
                    border.color: "#000000"
                    Text {
                        anchors.centerIn: parent
                        text: "Done"
                        font.family: fontSansSemi.name
                        font.pixelSize: 30 * root.ps
                        color: doneArea.pressed ? "#000000" : "#FFFFFF"
                    }
                    MouseArea { id: doneArea; anchors.fill: parent; onClicked: root.oskPress("done") }
                }
            }
        }
    }

    // Dev drive bridge: file-based semantic commands with screenshot acks,
    // so design gets verified without a human driving the emulator. Runs
    // only when /tmp/chiron-drive/enable exists - never on the tablet.
    Timer {
        id: drive
        interval: 300; repeat: true; running: false
        property int lastSeq: 0
        onTriggered: {
            // HTTP, not file polling: Qt caches file XHRs after a couple of
            // reads, and the same channel can drive the real tablet later.
            const xhr = new XMLHttpRequest()
            xhr.onreadystatechange = function() {
                if (xhr.readyState !== XMLHttpRequest.DONE) return
                if (xhr.status !== 200 || !xhr.responseText) return
                try {
                    drive.exec(JSON.parse(xhr.responseText))
                } catch (e) {}
            }
            xhr.open("GET", root.serverBase + "/drive/next")
            xhr.send()
        }
        function exec(c) {
            var err = ""
            try {
                execInner(c)
            } catch (e) {
                err = e.toString()
                console.log("[drive] error on", c.cmd, ":", err)
            }
            // Ack through the server - completion must not depend on the
            // screenshot pipeline, which is best-effort evidence.
            const ack = new XMLHttpRequest()
            ack.open("POST", root.serverBase + "/drive/ack")
            ack.setRequestHeader("Content-Type", "application/json")
            ack.send(JSON.stringify({seq: c.seq, cmd: c.cmd, mode: root.mode,
                                     page: root.page, error: err}))
            ackTimer.seq = c.seq
            ackTimer.restart()
        }
        // The n-th item on the current page, for commands that omit ids.
        function driveItem(n) {
            const list = root.itemsForPage(root.page)
            return list.length > n ? list[n].item : ""
        }
        function execInner(c) {
            if (c.cmd === "dump")
                console.log("[drive]", JSON.stringify({mode: root.mode, page: root.page,
                    sel: root.selByItem, idk: root.idkByItem, conf: root.confByItem,
                    items: root.itemPages.length, server: root.serverBase,
                    w: root.width, h: root.height, ps: root.ps,
                    pw: pageImage.paintedWidth, ph: pageImage.paintedHeight,
                    px0: root.px0, py0: root.py0}))
            else if (c.cmd === "level") root.setMap("selByItem", root.screener ? root.screener.item_id : "", c.i)
            else if (c.cmd === "select") root.tapLetter(c.item || driveItem(0), c.i)
            else if (c.cmd === "conf") root.tapConf(c.item || driveItem(0), c.v)
            else if (c.cmd === "idk") {
                if (c.on === false) root.setMap("idkByItem", c.item || driveItem(c.slot || 0), false)
                else root.tapIdk(c.item || driveItem(c.slot || 0))
            }
            else if (c.cmd === "page") {
                if (root.mode === "results") root.resultsPage = c.n
                else root.page = c.n
            }
            else if (c.cmd === "checkin") root.checkIn()
            else if (c.cmd === "ink" && c.stroke) { root.strokesForPage(root.page).push(c.stroke); ink.requestFull() }
            else if (c.cmd === "next") root.advance()
            else if (c.cmd === "contents") root.openContents()
            else if (c.cmd === "type") {
                const tid = c.item || driveItem(c.slot || 0)
                root.setMap("textByItem", tid, c.text || "")
                if ((c.text || "") !== "") root.setMap("idkByItem", tid, false)
            }
            else if (c.cmd === "kb") root.openKb(c.item || driveItem(c.slot || 0))
            else if (c.cmd === "take5") root.takeBreak()
            else if (c.cmd === "resume") root.resumeFromBreak()
            else if (c.cmd === "breaktest") {
                // Harness-only: stage a suggestion without waiting 22 minutes.
                root.breakSuggestion = { minutes: 5, kind: "short",
                    note: "Chunk done. Five minutes, eyes off screens." }
                root.mode = "breakSuggested"
            }
            else if (c.cmd === "goto") {
                var row = null
                if (root.contentsMeta)
                    for (var gi = 0; gi < root.contentsMeta.rows.length; gi++)
                        if (root.contentsMeta.rows[gi].unit === c.unit)
                            row = root.contentsMeta.rows[gi]
                root.contentsGoto(row)
            }
            else if (c.cmd === "server") { root.serverBase = c.url; root.bootstrapTried = false; root.loadMeta() }
            else if (c.cmd === "reload") { root.bootstrapTried = false; root.loadMeta() }
            // "shot" and unknown commands just ack with a screenshot
        }
    }
    Timer {
        id: ackTimer
        interval: 700
        property int seq: 0
        onTriggered: {
            const ok = root.grabToImage(function(res) {
                const saved = res.saveToFile("/tmp/chiron-drive/state-" + ackTimer.seq + ".png")
                console.log("[drive] ack", ackTimer.seq, "saved:", saved)
            })
            if (!ok) console.log("[drive] grabToImage refused for seq", ackTimer.seq)
        }
    }

    // Results view: server-rendered pages (headline, gate bar, per-item
    // reveals with typeset math). Native contributes only the page turns
    // and the action button, whose label the server drives.
    function advance() {
        closeKb()
        inkByPage = ({}); idkByItem = ({}); selByItem = ({}); confByItem = ({})
        textByItem = ({})
        checkinResult = null
        resultsMeta = null
        resultsPage = 0
        browseUnit = ""
        savedPage = 0
        if (breakSuggestion !== null) {
            // The pacing system asked for a pause; the break screen decides
            // before the next chapter opens.
            mode = "breakSuggested"
            return
        }
        loadMeta()
    }

    function doReset() {
        mode = "loading"
        const xhr = new XMLHttpRequest()
        xhr.onreadystatechange = function() {
            if (xhr.readyState !== XMLHttpRequest.DONE) return
            if (xhr.status === 200) {
                inkByPage = ({}); idkByItem = ({}); selByItem = ({}); confByItem = ({})
                textByItem = ({})
                checkinResult = null; resultsMeta = null; resultsPage = 0
                contentsMeta = null; browseUnit = ""; homeUnit = ""
                savedPage = 0; breakSuggestion = null
                bootstrapTried = false
                loadMeta()
            } else {
                errorText = "Reset failed (" + xhr.status + ")"
                mode = "error"
            }
        }
        xhr.open("POST", tok(serverBase + "/reset"))
        xhr.setRequestHeader("Content-Type", "application/json")
        xhr.send(JSON.stringify({ subject: subject, confirm: true }))
    }

    function takeBreak() {
        breakStartMs = Date.now()
        mode = "breakActive"
    }
    function resumeFromBreak() {
        if (mode === "breakActive" && breakStartMs > 0)
            pendingBreakMinutes += (Date.now() - breakStartMs) / 60000
        breakSuggestion = null
        breakStartMs = 0
        loadMeta()
    }

    Image {
        id: resultsImage
        visible: root.mode === "results" && root.resultsMeta !== null
        anchors.fill: parent
        fillMode: Image.PreserveAspectFit
        source: root.mode === "results" && root.resultsMeta !== null
            ? root.tok(root.serverBase + "/pages/" + root.subject + "/results/"
              + root.resultsPage + "?v=" + root.resultsMeta.hash)
            : ""
        asynchronous: true
        cache: true
    }
    property real rx0: resultsImage.x + (resultsImage.width - resultsImage.paintedWidth) / 2
    property real ry0: resultsImage.y + (resultsImage.height - resultsImage.paintedHeight) / 2
    Text {
        // A graded check-in without typeset pages should not strand the
        // learner: state the fact plainly and let the action move on.
        visible: root.mode === "results" && root.resultsMeta === null
        text: "Checked in."
        font.family: fontSerif.name
        font.pixelSize: 32 * root.ps
        anchors.centerIn: parent
    }
    PageTurn {
        visible: root.mode === "results" && root.resultsMeta !== null
                 && root.resultsMeta.count > 1
        glyph: "‹"
        active: root.resultsPage > 0
        x: root.rx0 + 110 * root.ps
        y: root.ry0 + 2078 * root.ps
        onTapped: root.resultsPage--
    }
    PageTurn {
        visible: root.mode === "results" && root.resultsMeta !== null
                 && root.resultsMeta.count > 1
        glyph: "›"
        active: root.resultsMeta !== null
                && root.resultsPage < root.resultsMeta.count - 1
        x: root.rx0 + 186 * root.ps
        y: root.ry0 + 2078 * root.ps
        onTapped: root.resultsPage++
    }
    ActionButton {
        visible: root.mode === "results"
        label: root.resultsMeta && root.resultsMeta.action
               ? root.resultsMeta.action : "Next chapter"
        x: root.rx0 + 1510 * root.ps - width
        y: root.ry0 + 2078 * root.ps
        onTapped: root.advance()
    }
    QuietButton {
        visible: root.mode === "results"
        label: "Close"
        x: root.rx0 + 262 * root.ps
        y: root.ry0 + 2078 * root.ps
        onTapped: root.close()
    }

    // Break screens (SPEC §6): suggested offers the pause, active is a
    // still page until any tap resumes. Same static skeleton as the waits.
    Item {
        visible: root.mode === "breakSuggested" || root.mode === "breakActive"
        anchors.fill: parent
        property bool suggested: root.mode === "breakSuggested"

        MouseArea {
            // Any tap resumes an active break; declared first so the
            // suggested-state buttons stack above it.
            anchors.fill: parent
            enabled: root.mode === "breakActive"
            onClicked: root.resumeFromBreak()
        }
        Text {
            id: breakDinkus
            x: root.nx0 + (1620 * root.ps - width) / 2
            y: root.ny0 + 880 * root.ps
            text: "✱ ✱ ✱"
            font.family: fontSerif.name
            font.pixelSize: 36 * root.ps
            font.letterSpacing: 26 * root.ps
            color: "#000000"
        }
        Text {
            id: breakStatement
            x: root.nx0 + (1620 * root.ps - width) / 2
            y: breakDinkus.y + breakDinkus.height + 56 * root.ps
            text: parent.suggested ? "A good place to pause." : "On break."
            font.family: fontSerifIt.name
            font.italic: true
            font.pixelSize: 40 * root.ps
            color: "#000000"
        }
        Text {
            id: breakSub
            x: root.nx0 + (1620 * root.ps - width) / 2
            y: breakStatement.y + breakStatement.height + 14 * root.ps
            width: 1100 * root.ps
            horizontalAlignment: Text.AlignHCenter
            wrapMode: Text.Wrap
            text: parent.suggested
                  ? (root.breakSuggestion ? root.breakSuggestion.note : "")
                  : "Tap anywhere when you're ready."
            font.family: fontSerif.name
            font.pixelSize: 28 * root.ps
            lineHeightMode: Text.FixedHeight
            lineHeight: 40 * root.ps
            color: "#666666"
        }
        Row {
            visible: parent.suggested
            spacing: 24 * root.ps
            x: root.nx0 + (1620 * root.ps - width) / 2
            y: breakSub.y + breakSub.height + 72 * root.ps
            ActionButton {
                label: root.breakSuggestion && root.breakSuggestion.kind === "long"
                       ? "Take fifteen" : "Take five"
                onTapped: root.takeBreak()
            }
            QuietButton {
                label: "Keep reading"
                onTapped: root.resumeFromBreak()
            }
        }
        QuietButton {
            label: "Close"
            x: root.nx0 + 110 * root.ps
            y: root.ny0 + 2078 * root.ps
            onTapped: root.close()
        }
    }

    // Contents / spine: the rendered page plus invisible row tap targets
    // and the way back.
    Image {
        visible: root.mode === "contents" && root.contentsMeta !== null
        anchors.fill: parent
        fillMode: Image.PreserveAspectFit
        source: root.mode === "contents" && root.contentsMeta !== null
            ? root.tok(root.serverBase + "/pages/" + root.subject + "/contents/0?v="
              + root.contentsMeta.hash)
            : ""
        asynchronous: true
        cache: true
    }
    MouseArea {
        visible: root.mode === "contents" && root.contentsMeta !== null
        anchors.fill: parent
        onClicked: (e) => {
            const m = root.contentsMeta
            if (m === null) return
            const py = (e.y - root.ny0) / root.ps
            const i = Math.floor((py - m.rows_top) / m.row_h)
            if (i >= 0 && i < m.rows.length) root.contentsGoto(m.rows[i])
        }
    }
    ActionButton {
        visible: root.mode === "contents"
        label: "Back to the chapter"
        x: root.nx0 + 1510 * root.ps - width
        y: root.ny0 + 2078 * root.ps
        onTapped: root.contentsGoto({ unit: root.homeUnit, state: "in_progress" })
    }
    QuietButton {
        id: contentsCloseBtn
        // The discoverable way out (the AppLoad top-edge drag gesture is
        // not something anyone finds on their own).
        visible: root.mode === "contents"
        label: "Close Chiron"
        x: root.nx0 + 110 * root.ps
        y: root.ny0 + 2078 * root.ps
        onTapped: root.close()
    }
    QuietButton {
        visible: root.mode === "contents"
        label: "Start over"
        x: contentsCloseBtn.x + contentsCloseBtn.width + 24 * root.ps
        y: root.ny0 + 2078 * root.ps
        onTapped: root.mode = "confirmReset"
    }

    // Starting over asks first, and says what actually happens: the old
    // book is archived on the server, never deleted.
    Item {
        visible: root.mode === "confirmReset"
        anchors.fill: parent
        Text {
            id: resetDinkus
            x: root.nx0 + (1620 * root.ps - width) / 2
            y: root.ny0 + 880 * root.ps
            text: "✱ ✱ ✱"
            font.family: fontSerif.name
            font.pixelSize: 36 * root.ps
            font.letterSpacing: 26 * root.ps
            color: "#000000"
        }
        Text {
            id: resetStatement
            x: root.nx0 + (1620 * root.ps - width) / 2
            y: resetDinkus.y + resetDinkus.height + 56 * root.ps
            text: "Start the book over?"
            font.family: fontSerifIt.name
            font.italic: true
            font.pixelSize: 40 * root.ps
            color: "#000000"
        }
        Text {
            id: resetSub
            x: root.nx0 + (1620 * root.ps - width) / 2
            y: resetStatement.y + resetStatement.height + 14 * root.ps
            width: 1100 * root.ps
            horizontalAlignment: Text.AlignHCenter
            wrapMode: Text.Wrap
            text: "Calibration runs again and a new book is written for where you are now. The current book is archived on the server - nothing is deleted."
            font.family: fontSerif.name
            font.pixelSize: 28 * root.ps
            lineHeightMode: Text.FixedHeight
            lineHeight: 40 * root.ps
            color: "#666666"
        }
        Row {
            spacing: 24 * root.ps
            x: root.nx0 + (1620 * root.ps - width) / 2
            y: resetSub.y + resetSub.height + 72 * root.ps
            ActionButton {
                label: "Start over"
                onTapped: root.doReset()
            }
            QuietButton {
                label: "Keep reading"
                onTapped: root.mode = "contents"
            }
        }
    }

    // Waits and errors (SPEC §6): a shared static skeleton - dinkus,
    // italic statement, sub-line; errors add [Try again] and a detail
    // line. Nothing on these screens moves, blinks, or counts.
    property bool authoring: false
    property bool authoringLong: false
    Timer {
        id: authoringLongTimer
        interval: 45000
        onTriggered: root.authoringLong = true
    }

    Item {
        visible: root.mode === "loading" || root.mode === "submitting"
                 || root.mode === "error"
        anchors.fill: parent

        property bool isError: root.mode === "error"
        property bool isAuthoringErr: root.errorText.indexOf("authoring") !== -1

        Text {
            id: dinkus
            x: root.nx0 + (1620 * root.ps - width) / 2
            y: root.ny0 + 880 * root.ps
            text: parent.isError ? "✱" : "✱ ✱ ✱"
            font.family: fontSerif.name
            font.pixelSize: 36 * root.ps
            font.letterSpacing: 26 * root.ps
            color: parent.isError ? "#8A3D30" : "#000000"
        }
        Text {
            id: waitStatement
            x: root.nx0 + (1620 * root.ps - width) / 2
            y: dinkus.y + dinkus.height + 56 * root.ps
            text: {
                if (parent.isError)
                    return parent.isAuthoringErr ? "This chapter couldn't be written."
                                                 : "The server can't be reached."
                if (root.mode === "submitting") return "Grading your answers."
                if (root.authoring) return "The next chapter is being written."
                return "Opening the book."
            }
            font.family: fontSerifIt.name
            font.italic: true
            font.pixelSize: 40 * root.ps
            color: "#000000"
        }
        Text {
            id: waitSub
            x: root.nx0 + (1620 * root.ps - width) / 2
            y: waitStatement.y + waitStatement.height + 14 * root.ps
            width: 1100 * root.ps
            horizontalAlignment: Text.AlignHCenter
            wrapMode: Text.Wrap
            text: {
                if (parent.isError)
                    return parent.isAuthoringErr
                        ? "Nothing is lost — your results are kept. Try again, or come back later."
                        : "Your work is saved on this tablet. Nothing is lost."
                if (root.mode === "submitting") return "A few seconds."
                if (root.authoring)
                    return "Shaped by today's answers. It is usually ready before you finish reading the results."
                return ""
            }
            font.family: fontSerif.name
            font.pixelSize: 28 * root.ps
            lineHeightMode: Text.FixedHeight
            lineHeight: 40 * root.ps
            color: "#666666"
        }
        Text {
            visible: !parent.isError && root.authoring && root.authoringLong
            x: root.nx0 + (1620 * root.ps - width) / 2
            y: waitSub.y + waitSub.height + 14 * root.ps
            text: "Still writing — Chiron checks every few seconds."
            font.family: fontSerif.name
            font.pixelSize: 26 * root.ps
            color: "#777777"
        }
        ActionButton {
            id: tryAgainBtn
            visible: parent.isError
            label: "Try again"
            x: root.nx0 + (1620 * root.ps - width) / 2
            y: waitSub.y + waitSub.height + 72 * root.ps
            onTapped: root.loadMeta()
        }
        Text {
            visible: parent.isError
            x: root.nx0 + (1620 * root.ps - width) / 2
            y: tryAgainBtn.y + tryAgainBtn.height + 52 * root.ps
            text: (parent.isAuthoringErr ? "AUTHORING FAILED · " : "HOST UNREACHABLE · ")
                  + Qt.formatTime(new Date(), "hh:mm")
            font.family: fontSans.name
            font.pixelSize: 22 * root.ps
            font.letterSpacing: 22 * 0.06 * root.ps
            color: "#777777"
        }
        QuietButton {
            label: "Close"
            x: root.nx0 + 110 * root.ps
            y: root.ny0 + 2078 * root.ps
            onTapped: root.close()
        }
    }
}
