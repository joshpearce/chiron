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

    // "loading" | "reading" | "error"
    property string mode: "loading"
    property string errorText: ""
    property string chapterTitle: ""
    property string pagesHash: ""
    property int pageCount: 0
    property int page: 0

    function loadMeta() {
        mode = "loading"
        const xhr = new XMLHttpRequest()
        xhr.onreadystatechange = function() {
            if (xhr.readyState !== XMLHttpRequest.DONE) return
            if (xhr.status === 200) {
                const m = JSON.parse(xhr.responseText)
                chapterTitle = m.title
                pageCount = m.count
                pagesHash = m.hash
                page = 0
                mode = "reading"
                console.log("[chiron] chapter:", m.unit, m.count, "pages")
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

    // Loading / error states
    Column {
        anchors.centerIn: parent
        spacing: 24
        visible: root.mode !== "reading"

        Text {
            text: "Chiron"
            font.pixelSize: 64
            font.family: "serif"
            anchors.horizontalCenter: parent.horizontalCenter
        }
        Text {
            text: root.mode === "loading" ? "fetching chapter..." : root.errorText
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
