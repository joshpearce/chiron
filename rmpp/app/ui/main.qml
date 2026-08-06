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

    // Page-turn tap zones
    MouseArea {
        visible: root.mode === "reading"
        anchors { left: parent.left; top: parent.top; bottom: parent.bottom }
        width: parent.width * 0.25
        onClicked: if (root.page > 0) root.page--
    }
    MouseArea {
        visible: root.mode === "reading"
        anchors { right: parent.right; top: parent.top; bottom: parent.bottom }
        width: parent.width * 0.25
        onClicked: if (root.page < root.pageCount - 1) root.page++
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
