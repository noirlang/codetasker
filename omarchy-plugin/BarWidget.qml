import QtQuick 2.15
import QtQuick.Layouts 1.15
import QtQuick.Controls 2.15
import "CodeTaskerApi.js" as Api

Rectangle {
    id: root
    implicitWidth: contentRow.implicitWidth + 14
    implicitHeight: 28
    radius: 4
    color: mouseArea.containsMouse ? "#1a1a1a" : "transparent"

    property int unreadCount: 0
    property string serverUrl: "http://localhost:8080"
    property string appToken: ""

    Component.onCompleted: {
        refreshSettingsAndUnread()
        pollTimer.start()
    }

    function refreshSettingsAndUnread() {
        serverUrl = Api.getSetting("serverUrl", "http://localhost:8080")
        appToken = Api.getSetting("appToken", "")

        if (!appToken) {
            unreadCount = 0
            return
        }

        Api.fetchUnreadCount(serverUrl, appToken, function(err, count) {
            if (!err) {
                unreadCount = count
            }
        })
    }

    Timer {
        id: pollTimer
        interval: 10000 // Refresh every 10 seconds
        repeat: true
        running: true
        onTriggered: refreshSettingsAndUnread()
    }

    RowLayout {
        id: contentRow
        anchors.centerIn: parent
        spacing: 5

        Text {
            text: "</>"
            font.bold: true
            font.pixelSize: 13
            font.family: "Monospace"
            color: unreadCount > 0 ? "#ef4444" : (appToken ? "#ffffff" : "#666666")
        }

        Rectangle {
            visible: unreadCount > 0
            implicitWidth: Math.max(14, countText.implicitWidth + 6)
            implicitHeight: 14
            radius: 7
            color: "#ef4444"

            Text {
                id: countText
                anchors.centerIn: parent
                text: unreadCount > 99 ? "99+" : unreadCount.toString()
                color: "#ffffff"
                font.pixelSize: 9
                font.bold: true
            }
        }
    }

    MouseArea {
        id: mouseArea
        anchors.fill: parent
        hoverEnabled: true
        cursorShape: Qt.PointingHandCursor
        onClicked: {
            refreshSettingsAndUnread()
            if (typeof shell !== "undefined" && shell.togglePanel) {
                shell.togglePanel("codetasker.notifications")
            } else if (parent && parent.togglePanel) {
                parent.togglePanel()
            }
        }
    }
}
