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
    // Non-null when the chapter is the placement screener: rendered natively,
    // the question box is the UI.
    property var screener: null
    property var checkinResult: null
    // Pages the learner explicitly marked "I don't know" - wins over ink.
    property var idkPages: ({})
    // Structured selections (screener rating, MCQ option) per page.
    property var selByPage: ({})
    // Confidence per page, 1-4; unset means shaky.
    property var confByPage: ({})

    // Copy-on-write: assigning the SAME object reference back to a var
    // property does not signal a change, so bindings (button checked state,
    // Check in enabled) never re-evaluate. A fresh object every write does.
    function setMap(name, p, v) {
        var old = root[name]
        var m = {}
        for (var k in old) m[k] = old[k]
        m[p] = v
        root[name] = m
    }

    function itemForPage(p) {
        for (var i = 0; i < itemPages.length; i++)
            if (itemPages[i].page === p) return itemPages[i]
        return null
    }

    function loadMeta() {
        mode = "loading"
        const xhr = new XMLHttpRequest()
        xhr.onreadystatechange = function() {
            if (xhr.readyState !== XMLHttpRequest.DONE) return
            if (xhr.status === 200) {
                const m = JSON.parse(xhr.responseText)
                chapterTitle = m.title
                chapterUnit = m.unit
                pageCount = m.count
                pagesHash = m.hash
                itemPages = m.items || []
                screener = m.screener || null
                page = 0
                mode = screener ? "screener" : "reading"
                console.log("[chiron] chapter:", m.unit, m.count, "pages,",
                            itemPages.length, "answer pages")
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

    Component.onCompleted: loadMeta()

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
                        checked: root.selByPage[0] === index
                        onClicked: root.setMap("selByPage", 0, index)
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
            enabled: root.selByPage[0] !== undefined && root.selByPage[0] !== null
            anchors.horizontalCenter: parent.horizontalCenter
            onClicked: root.checkIn()
        }
    }

    function checkIn() {
        mode = "submitting"
        const items = []
        if (screener) {
            items.push({ item_id: screener.item_id,
                         selected_index: selByPage[0],
                         strokes: [] })
        } else
        for (var i = 0; i < itemPages.length; i++) {
            const it = itemPages[i]
            const entry = { item_id: it.id,
                            idk: idkPages[it.page] === true,
                            strokes: strokesForPage(it.page) }
            if (selByPage[it.page] !== undefined && selByPage[it.page] !== null)
                entry.selected_index = selByPage[it.page]
            if (confByPage[it.page])
                entry.confidence = confByPage[it.page]
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
                    inkByPage = ({}); idkPages = ({}); selByPage = ({}); confByPage = ({})
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

    // Structured answer controls, docked above the footer on answer pages.
    Row {
        id: controlBar
        visible: root.mode === "reading" && root.itemForPage(root.page) !== null
        spacing: 10
        anchors { horizontalCenter: parent.horizontalCenter; bottom: parent.bottom; bottomMargin: 44 }

        property var item: root.itemForPage(root.page)

        // Screener: rate yourself 1-5.
        Repeater {
            model: controlBar.item && controlBar.item.check === "screener" ? 5 : 0
            delegate: Button {
                text: (index + 1)
                width: 64
                font.pixelSize: 24
                checkable: true
                checked: root.selByPage[root.page] === index
                onClicked: root.setMap("selByPage", root.page, index)
            }
        }
        // MCQ: one lettered button per printed option.
        Repeater {
            model: controlBar.item && controlBar.item.kind === "mcq"
                   ? controlBar.item.options : 0
            delegate: Button {
                text: String.fromCharCode(65 + index)
                width: 64
                font.pixelSize: 24
                checkable: true
                checked: root.selByPage[root.page] === index
                onClicked: root.setMap("selByPage", root.page, index)
            }
        }
        // Constructed items: confidence in the ink answer.
        Repeater {
            model: controlBar.item && controlBar.item.kind !== "mcq"
                   && controlBar.item.check !== "screener"
                   ? ["unsure", "shaky", "confident", "sure"] : 0
            delegate: Button {
                text: modelData
                font.pixelSize: 18
                checkable: true
                checked: root.confByPage[root.page] === index + 1
                onClicked: root.setMap("confByPage", root.page, index + 1)
            }
        }
    }

    // "I don't know" toggle, shown only on answer pages. Explicit beats
    // inferred: a tick drawn in ink once came back graded correct because
    // the vision model answered the question itself.
    Button {
        visible: root.mode === "reading" && root.itemForPage(root.page) !== null
        checkable: true
        checked: root.idkPages[root.page] === true
        text: checked ? "✓ I don't know" : "I don't know"
        font.pixelSize: 20
        anchors { right: parent.right; top: parent.top; margins: 10 }
        onToggled: {
            var m = root.idkPages
            m[root.page] = checked
            root.idkPages = m
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
                    root.idkPages = ({})
                    root.selByPage = ({})
                    root.confByPage = ({})
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
            text: root.mode === "loading" ? "fetching chapter..."
                : root.mode === "submitting" ? "Grading your answers - the next chapter is being written..."
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
