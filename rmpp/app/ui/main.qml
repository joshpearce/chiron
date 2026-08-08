import QtQuick 2.15
import QtQuick.Controls 2.15

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
        border.color: pill.muted && !pill.selected ? "#BBBBBB" : "#000000"
        Text {
            id: pillText
            anchors.centerIn: parent
            text: pill.label
            font.family: fontSansSemi.name
            font.pixelSize: 26 * root.ps
            color: pill.selected !== pillArea.pressed ? "#FFFFFF"
                 : (pill.muted ? "#BBBBBB" : "#000000")
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
        border.color: lb.muted && !lb.selected ? "#BBBBBB" : "#000000"
        Text {
            anchors.centerIn: parent
            text: lb.letter
            font.family: fontSansBold.name
            font.pixelSize: 30 * root.ps
            color: lb.selected !== lbArea.pressed ? "#FFFFFF"
                 : (lb.muted ? "#BBBBBB" : "#000000")
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
        border.color: ab.active ? "#000000" : "#BBBBBB"
        Text {
            id: abText
            anchors.centerIn: parent
            text: ab.label
            font.family: fontSansSemi.name
            font.pixelSize: 28 * root.ps
            color: ab.active ? (abArea.pressed ? "#FFFFFF" : "#000000") : "#999999"
        }
        MouseArea {
            id: abArea
            anchors.fill: parent
            onClicked: if (ab.active) ab.tapped()
        }
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
        border.color: pt.active ? "#666666" : "#CCCCCC"
        Text {
            anchors.centerIn: parent
            text: pt.glyph
            font.family: fontSans.name
            font.pixelSize: 34 * root.ps
            color: pt.active ? (ptArea.pressed ? "#FFFFFF" : "#333333") : "#BBBBBB"
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
    // Answer state keyed by item id - pages hold several items now.
    property var idkByItem: ({})
    property var selByItem: ({})
    property var confByItem: ({})

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

    property string loadingWhy: "fetching chapter..."

    Timer {
        id: retryMeta
        interval: 3000
        onTriggered: root.loadMeta()
    }

    function loadMeta() {
        mode = "loading"
        const xhr = new XMLHttpRequest()
        xhr.onreadystatechange = function() {
            if (xhr.readyState !== XMLHttpRequest.DONE) return
            if (xhr.status === 200) {
                const m = JSON.parse(xhr.responseText)
                if (m.authoring) {
                    // Grades came back instantly; the chapter is still being
                    // written. Keep the learner informed and poll.
                    loadingWhy = "The next chapter is being written..."
                    retryMeta.restart()
                    return
                }
                if (m.authoring_error) {
                    errorText = "Chapter authoring failed:\n" + m.authoring_error
                    mode = "error"
                    return
                }
                loadingWhy = "fetching chapter..."
                chapterTitle = m.title
                chapterUnit = m.unit
                pageCount = m.count
                pagesHash = m.hash
                itemPages = m.items || []
                screener = m.screener || null
                layoutC = m.layout || null
                page = 0
                mode = screener ? "screener" : "reading"
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
                boot.open("POST", serverBase + "/exchange")
                boot.setRequestHeader("Content-Type", "application/json")
                boot.send(JSON.stringify({ subject: subject, phase: "start" }))
            } else {
                errorText = "No chapter available (" + xhr.status + ").\n"
                          + "Server: " + serverBase
                mode = "error"
                console.log("[chiron] meta failed:", xhr.status)
            }
        }
        xhr.open("GET", serverBase + "/pages/" + subject)
        xhr.send()
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
        anchors.fill: parent
        visible: root.mode === "reading"
        fillMode: Image.PreserveAspectFit
        // The hash pins the URL to the chapter revision, so stale cached
        // pages can never show for a regenerated chapter.
        source: root.mode === "reading"
            ? root.serverBase + "/pages/" + root.subject + "/" + root.page
              + "?v=" + root.pagesHash
            : ""
        asynchronous: true
        cache: true
    }

    Canvas {
        id: ink
        anchors.fill: pageImage
        visible: root.mode === "reading"

        property var liveStroke: null

        onPaint: {
            const ctx = getContext("2d")
            ctx.clearRect(0, 0, width, height)
            ctx.strokeStyle = "black"
            ctx.lineWidth = 3
            ctx.lineCap = "round"
            ctx.lineJoin = "round"
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
        }

        MouseArea {
            anchors.fill: parent
            function endStroke() {
                if (ink.liveStroke && ink.liveStroke.length > 1)
                    root.strokesForPage(root.page).push(ink.liveStroke)
                ink.liveStroke = null
                ink.requestPaint()
            }
            onPressed: (e) => {
                if (!root.inkAllowedAt(e.x / ink.width, e.y / ink.height)) return
                ink.liveStroke = [{x: e.x / ink.width, y: e.y / ink.height}]
                ink.requestPaint()
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
                ink.requestPaint()
            }
            onReleased: endStroke()
        }
    }

    onPageChanged: ink.requestPaint()

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
        visible: root.mode === "reading" && root.page === root.pageCount - 1
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

    // Native placement screen: the question box holds the level buttons.
    Column {
        visible: root.mode === "screener"
        anchors { top: parent.top; horizontalCenter: parent.horizontalCenter; topMargin: 60 }
        width: Math.min(parent.width * 0.86, 980)
        spacing: 28

        Text {
            width: parent.width
            text: root.screener ? root.screener.intro : ""
            font.pixelSize: 22
            font.family: "serif"
            wrapMode: Text.Wrap
        }
        Rectangle {
            width: parent.width
            height: screenerBox.height + 56
            color: "white"
            border.color: "black"
            border.width: 3

            Column {
                id: screenerBox
                anchors { top: parent.top; left: parent.left; right: parent.right; margins: 28 }
                spacing: 18

                Text {
                    width: parent.width
                    text: root.screener ? root.screener.prompt : ""
                    font.pixelSize: 24
                    font.family: "serif"
                    wrapMode: Text.Wrap
                }
                Repeater {
                    model: root.screener ? root.screener.options : []
                    delegate: Button {
                        width: screenerBox.width
                        height: 56
                        checkable: true
                        checked: root.screener && root.selByItem[root.screener.item_id] === index
                        onClicked: root.setMap("selByItem", root.screener.item_id, index)
                        contentItem: Row {
                            spacing: 16
                            leftPadding: 12
                            Text {
                                text: (index + 1) + "."
                                font.pixelSize: 22
                                font.bold: true
                                anchors.verticalCenter: parent.verticalCenter
                            }
                            Text {
                                text: modelData
                                font.pixelSize: 22
                                anchors.verticalCenter: parent.verticalCenter
                            }
                        }
                    }
                }
            }
        }
        Button {
            text: "Check in"
            font.pixelSize: 24
            enabled: root.screener && root.selByItem[root.screener.item_id] !== undefined
            anchors.horizontalCenter: parent.horizontalCenter
            onClicked: root.checkIn()
        }
    }

    // Pen input is captured only inside a constructed item's box, above the
    // shelf rule; MCQ boxes take no ink. Pages without published items
    // (prose, beat boxes) are open ink room.
    function inkAllowedAt(nx, ny) {
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
        // error would ship stale item pages from a previous chapter.
        if (mode !== "reading" && mode !== "screener") return
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
                    checkinResult = null
                    loadMeta()
                } else {
                    resultsMeta = checkinResult.results_pages || null
                    resultsPage = 0
                    mode = "results"
                }
            } else {
                errorText = "Check-in failed (" + xhr.status + "): "
                          + xhr.responseText.substring(0, 200)
                mode = "error"
            }
        }
        xhr.open("POST", serverBase + "/ink/" + subject)
        xhr.setRequestHeader("Content-Type", "application/json")
        xhr.send(JSON.stringify({ unit: chapterUnit, items: items }))
    }

    // Per-item controls mapped into each answer box's reserved strip
    // (SPEC §2.3; regions come from the pages meta - the server enforces
    // the same geometry it publishes). Constructed: one centered row, IDK
    // left, confidence right. MCQ: letters left + IDK right, then a
    // confidence row, both right-group aligned. Side insets 30 from the
    // box's inner edges.
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
                selected: ik
                x: mcq ? parent.width - width - 30 * root.ps : 30 * root.ps
                y: mcq ? 26 * root.ps : (parent.height - height) / 2
                onTapped: root.tapIdk(it.item)
            }
            Row {
                visible: mcq
                spacing: 12 * root.ps
                x: 30 * root.ps
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
            else if (c.cmd === "ink" && c.stroke) { root.strokesForPage(root.page).push(c.stroke); ink.requestPaint() }
            else if (c.cmd === "next") root.advance()
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
        inkByPage = ({}); idkByItem = ({}); selByItem = ({}); confByItem = ({})
        checkinResult = null
        resultsMeta = null
        resultsPage = 0
        loadMeta()
    }

    Image {
        id: resultsImage
        visible: root.mode === "results" && root.resultsMeta !== null
        anchors.fill: parent
        fillMode: Image.PreserveAspectFit
        source: root.mode === "results" && root.resultsMeta !== null
            ? root.serverBase + "/pages/" + root.subject + "/results/"
              + root.resultsPage + "?v=" + root.resultsMeta.hash
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

    // Loading / error states
    Column {
        anchors.centerIn: parent
        spacing: 24
        visible: root.mode === "loading" || root.mode === "submitting" || root.mode === "error"

        Text {
            text: "Chiron"
            font.pixelSize: 64
            font.family: "serif"
            anchors.horizontalCenter: parent.horizontalCenter
        }
        Text {
            text: root.mode === "loading" ? root.loadingWhy
                : root.mode === "submitting" ? "Grading your answers..."
                : root.errorText
            font.pixelSize: 24
            horizontalAlignment: Text.AlignHCenter
            anchors.horizontalCenter: parent.horizontalCenter
        }
        Button {
            visible: root.mode === "error"
            text: "Retry"
            font.pixelSize: 24
            anchors.horizontalCenter: parent.horizontalCenter
            onClicked: root.loadMeta()
        }
    }
}
