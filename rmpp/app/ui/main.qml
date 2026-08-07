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
            onPressed: (e) => {
                ink.liveStroke = [{x: e.x / ink.width, y: e.y / ink.height}]
                ink.requestPaint()
            }
            onPositionChanged: (e) => {
                if (!ink.liveStroke) return
                ink.liveStroke.push({x: e.x / ink.width, y: e.y / ink.height})
                ink.requestPaint()
            }
            onReleased: {
                if (ink.liveStroke && ink.liveStroke.length > 1)
                    root.strokesForPage(root.page).push(ink.liveStroke)
                ink.liveStroke = null
                ink.requestPaint()
            }
        }
    }

    onPageChanged: ink.requestPaint()

    // Page turning moved to explicit corner controls: the whole page
    // surface belongs to ink now.
    Row {
        visible: root.mode === "reading"
        spacing: 12
        anchors { left: parent.left; bottom: parent.bottom; margins: 10 }
        Button {
            text: "‹"
            font.pixelSize: 28
            width: 56
            enabled: root.page > 0
            onClicked: root.page--
        }
        Button {
            text: "›"
            font.pixelSize: 28
            width: 56
            enabled: root.page < root.pageCount - 1
            onClicked: root.page++
        }
    }

    // Progress footer, deliberately tiny
    Text {
        visible: root.mode === "reading"
        text: (root.page + 1) + " / " + root.pageCount
        font.pixelSize: 18
        color: "#666666"
        anchors { bottom: parent.bottom; horizontalCenter: parent.horizontalCenter; bottomMargin: 8 }
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

    // Strokes on a page belong to the item whose region contains their
    // first point, renormalized to the region so the transcriber gets a
    // tight crop instead of a mostly empty page.
    function strokesForItem(it) {
        const page = strokesForPage(it.page)
        const out = []
        for (var i = 0; i < page.length; i++) {
            const s = page[i]
            if (s.length === 0) continue
            if (s[0].y < it.top || s[0].y > it.top + it.h) continue
            const rs = []
            for (var j = 0; j < s.length; j++)
                rs.push({ x: s[j].x, y: (s[j].y - it.top) / it.h })
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
            const entry = { item_id: it.id,
                            idk: idkByItem[it.id] === true,
                            aspect: it.h > 0 ? 0.75 / it.h : 0.75,
                            strokes: strokesForItem(it) }
            if (selByItem[it.id] !== undefined && selByItem[it.id] !== null)
                entry.selected_index = selByItem[it.id]
            if (confByItem[it.id])
                entry.confidence = confByItem[it.id]
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

    // Per-item controls, one row mapped into each answer box's reserved
    // strip (regions come from the pages meta - the server enforces the
    // same geometry it publishes).
    Repeater {
        model: root.mode === "reading" && root.layoutC !== null
               ? root.itemsForPage(root.page) : []
        delegate: Row {
            spacing: 10
            property var it: modelData
            x: pageImage.x + (pageImage.width - pageImage.paintedWidth) / 2
               + pageImage.paintedWidth / 2 - width / 2
            y: pageImage.y + (pageImage.height - pageImage.paintedHeight) / 2
               + (it.top + it.h - (root.layoutC.strip_h / root.layoutC.page_h) / 2)
                 * pageImage.paintedHeight - height / 2

            Button {
                checkable: true
                checked: root.idkByItem[it.id] === true
                text: checked ? "✓ I don't know" : "I don't know"
                font.pixelSize: 14
                onToggled: root.setMap("idkByItem", it.id, checked)
            }
            Repeater {
                model: it.kind === "mcq" ? it.options : 0
                delegate: Button {
                    text: String.fromCharCode(65 + index)
                    width: 44
                    font.pixelSize: 15
                    checkable: true
                    checked: root.selByItem[it.id] === index
                    onClicked: root.setMap("selByItem", it.id, index)
                }
            }
            Repeater {
                model: it.kind !== "mcq" ? ["unsure", "shaky", "confident", "sure"] : 0
                delegate: Button {
                    text: modelData
                    font.pixelSize: 13
                    checkable: true
                    checked: root.confByItem[it.id] === index + 1
                    onClicked: root.setMap("confByItem", it.id, index + 1)
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
            return list.length > n ? list[n].id : ""
        }
        function execInner(c) {
            if (c.cmd === "dump")
                console.log("[drive]", JSON.stringify({mode: root.mode, page: root.page,
                    sel: root.selByItem, idk: root.idkByItem, conf: root.confByItem,
                    items: root.itemPages.length, server: root.serverBase}))
            else if (c.cmd === "level") root.setMap("selByItem", root.screener ? root.screener.item_id : "", c.i)
            else if (c.cmd === "select") root.setMap("selByItem", c.item || driveItem(0), c.i)
            else if (c.cmd === "conf") root.setMap("confByItem", c.item || driveItem(0), c.v)
            else if (c.cmd === "idk") root.setMap("idkByItem", c.item || driveItem(c.slot || 0), c.on === false ? false : true)
            else if (c.cmd === "page") root.page = c.n
            else if (c.cmd === "checkin") root.checkIn()
            else if (c.cmd === "ink" && c.stroke) { root.strokesForPage(root.page).push(c.stroke); ink.requestPaint() }
            else if (c.cmd === "next") { root.inkByPage = ({}); root.idkByItem = ({}); root.selByItem = ({}); root.confByItem = ({}); root.checkinResult = null; root.loadMeta() }
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

    // Check in, from the last page - answers up, grades and the next
    // chapter back.
    Button {
        visible: root.mode === "reading" && root.page === root.pageCount - 1
                 && root.itemPages.length > 0
        text: "Check in"
        font.pixelSize: 24
        anchors { right: parent.right; bottom: parent.bottom; margins: 10 }
        onClicked: root.checkIn()
    }

    // Results view
    Flickable {
        visible: root.mode === "results"
        anchors.fill: parent
        anchors.margins: 40
        contentHeight: resultsColumn.height
        clip: true

        Column {
            id: resultsColumn
            width: parent.width
            spacing: 16

            Text {
                text: {
                    if (!root.checkinResult || !root.checkinResult.gate) return "Checked in"
                    const g = root.checkinResult.gate
                    const pct = g.score !== undefined && g.score !== null
                        ? Math.round(g.score * 100) + "%" : ""
                    if (g.calibration) return "Calibration complete - " + pct
                    return (g.passed ? "Gate cleared - " : "Below the gate - ") + pct
                }
                font.pixelSize: 42
                font.family: "serif"
            }
            Repeater {
                model: root.checkinResult ? (root.checkinResult.results || []) : []
                delegate: Column {
                    width: resultsColumn.width
                    spacing: 2
                    Text {
                        width: parent.width
                        text: (modelData.verdict === "pass" ? "✓  " : "✗  ")
                              + modelData.item_id + "   "
                              + (root.checkinResult.transcripts
                                 ? "“" + (root.checkinResult.transcripts[modelData.item_id] || "") + "”"
                                 : "")
                        font.pixelSize: 22
                        wrapMode: Text.Wrap
                    }
                    Text {
                        width: parent.width
                        visible: modelData.feedback_md !== undefined && modelData.verdict !== "pass"
                        text: modelData.feedback_md || ""
                        font.pixelSize: 18
                        color: "#444444"
                        wrapMode: Text.Wrap
                    }
                }
            }
            Button {
                text: "Next chapter"
                font.pixelSize: 24
                onClicked: {
                    root.inkByPage = ({})
                    root.idkByItem = ({})
                    root.selByItem = ({})
                    root.confByItem = ({})
                    root.checkinResult = null
                    root.loadMeta()
                }
            }
        }
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
