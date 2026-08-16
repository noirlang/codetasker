import QtQuick
import QtQuick.Layouts
import QtQuick.Controls
import Quickshell
import Quickshell.Io
import qs.Commons
import qs.Ui
import "CodeTaskerApi.js" as Api

Panel {
  id: root
  moduleName: "codetasker.notifications"
  ipcTarget: "codetasker.notifications"
  manageIpc: false

  property var anchorItem: null
  property var hostWidget: null
  readonly property var barIdentity: hostWidget || root

  property string serverUrl: "https://codetasker.noirlang.tr"
  property string appToken: ""
  property bool isSetupMode: false
  property var notificationsList: []
  property bool isLoading: false
  property string errorMessage: ""

  property bool opened: false

  function open() {
    loadSettings()
    if (!appToken) {
      isSetupMode = true
    } else {
      loadNotifications()
    }
    opened = true
  }

  function close() {
    opened = false
  }

  function toggle() {
    if (opened) close()
    else open()
  }

  function loadSettings() {
    serverUrl = Api.getSetting("serverUrl", "https://codetasker.noirlang.tr")
    appToken = Api.getSetting("appToken", "")
    serverInput.text = serverUrl
    tokenInput.text = appToken
  }

  function saveSettings() {
    serverUrl = serverInput.text.trim() || "https://codetasker.noirlang.tr"
    appToken = tokenInput.text.trim()

    Api.setSetting("serverUrl", serverUrl)
    Api.setSetting("appToken", appToken)

    if (appToken) {
      isSetupMode = false
      if (hostWidget && typeof hostWidget.refresh === "function") hostWidget.refresh()
      loadNotifications()
    } else {
      errorMessage = "Please enter a valid App Token."
    }
  }

  function loadNotifications() {
    if (!appToken) return
    isLoading = true
    errorMessage = ""

    Api.fetchNotifications(serverUrl, appToken, function(err, items) {
      isLoading = false
      if (err) {
        errorMessage = err.message || "Failed to load notifications"
        notificationsList = []
      } else {
        notificationsList = items
        if (hostWidget && typeof hostWidget.refresh === "function") hostWidget.refresh()
      }
    })
  }

  function handleMarkAllRead() {
    Api.markAllRead(serverUrl, appToken, function(err) {
      if (!err) {
        loadNotifications()
      }
    })
  }

  function handleMarkRead(notifId) {
    Api.markRead(serverUrl, appToken, notifId, function(err) {
      if (!err) {
        loadNotifications()
      }
    })
  }

  KeyboardPanel {
    id: panel
    anchorItem: root.anchorItem
    owner: root.barIdentity
    bar: root.bar
    open: root.opened
    centerOnBar: true
    contentWidth: panel.fittedContentWidth(Style.space(380))
    contentHeight: panel.fittedContentHeight(Style.space(480))

    PanelKeyCatcher {
      id: keyCatcher
      anchors.fill: parent
      onCloseRequested: root.close()

      Rectangle {
        anchors.fill: parent
        color: "#0a0a0a"
        radius: 8
        border.color: "#262626"
        border.width: 1

        ColumnLayout {
          anchors.fill: parent
          anchors.margins: Style.space(16)
          spacing: Style.space(12)

          // ── Header ────────────────────────────────────────────────────────
          RowLayout {
            Layout.fillWidth: true
            spacing: 8

            Text {
              text: "</>"
              font.bold: true
              font.pixelSize: 15
              font.family: Style.font.family
              color: "#10b981"
            }

            Text {
              text: "CodeTasker Notifications"
              font.bold: true
              font.pixelSize: 13
              color: "#ffffff"
              font.family: Style.font.family
              Layout.fillWidth: true
            }

            // Unread badge
            Rectangle {
              visible: !isSetupMode && notificationsList.length > 0
              implicitWidth: Math.max(18, unreadText.implicitWidth + 8)
              implicitHeight: 18
              radius: 9
              color: "#ef4444"

              Text {
                id: unreadText
                anchors.centerIn: parent
                text: notificationsList.length.toString()
                color: "#ffffff"
                font.pixelSize: 10
                font.bold: true
              }
            }

            // Refresh icon
            Rectangle {
              visible: !isSetupMode
              implicitWidth: 26
              implicitHeight: 26
              radius: 4
              color: refreshMouse.containsMouse ? "#222222" : "#141414"

              Text {
                anchors.centerIn: parent
                text: "↻"
                color: "#a0a0a0"
                font.pixelSize: 12
              }

              MouseArea {
                id: refreshMouse
                anchors.fill: parent
                hoverEnabled: true
                cursorShape: Qt.PointingHandCursor
                onClicked: loadNotifications()
              }
            }

            // Settings gear
            Rectangle {
              implicitWidth: 26
              implicitHeight: 26
              radius: 4
              color: settingsMouse.containsMouse ? "#222222" : "#141414"

              Text {
                anchors.centerIn: parent
                text: "⚙"
                color: isSetupMode ? "#10b981" : "#a0a0a0"
                font.pixelSize: 12
              }

              MouseArea {
                id: settingsMouse
                anchors.fill: parent
                hoverEnabled: true
                cursorShape: Qt.PointingHandCursor
                onClicked: isSetupMode = !isSetupMode
              }
            }
          }

          Rectangle {
            Layout.fillWidth: true
            implicitHeight: 1
            color: "#222222"
          }

          // ── 1. SETUP MODE (Token Input Prompt on 1st run) ─────────────────
          ColumnLayout {
            visible: isSetupMode
            Layout.fillWidth: true
            Layout.fillHeight: true
            spacing: 12

            Text {
              text: "Connect to CodeTasker"
              font.bold: true
              font.pixelSize: 14
              color: "#ffffff"
              font.family: Style.font.family
            }

            Text {
              text: "Generate an App Token in CodeTasker (Settings → App Tokens) to receive notifications in Omarchy."
              wrapMode: Text.WordWrap
              Layout.fillWidth: true
              font.pixelSize: 11
              color: "#888888"
              font.family: Style.font.family
            }

            ColumnLayout {
              Layout.fillWidth: true
              spacing: 4

              Text {
                text: "SERVER URL"
                font.pixelSize: 9
                font.bold: true
                font.family: "Monospace"
                color: "#666666"
              }

              Rectangle {
                Layout.fillWidth: true
                implicitHeight: 32
                radius: 4
                color: "#121212"
                border.color: serverInput.activeFocus ? "#10b981" : "#2a2a2a"

                TextInput {
                  id: serverInput
                  anchors.fill: parent
                  anchors.margins: 6
                  text: "https://codetasker.noirlang.tr"
                  color: "#ffffff"
                  font.pixelSize: 11
                  verticalAlignment: TextInput.AlignVCenter
                }
              }
            }

            ColumnLayout {
              Layout.fillWidth: true
              spacing: 4

              Text {
                text: "APP TOKEN (ct_app_...)"
                font.pixelSize: 9
                font.bold: true
                font.family: "Monospace"
                color: "#666666"
              }

              Rectangle {
                Layout.fillWidth: true
                implicitHeight: 32
                radius: 4
                color: "#121212"
                border.color: tokenInput.activeFocus ? "#10b981" : "#2a2a2a"

                TextInput {
                  id: tokenInput
                  anchors.fill: parent
                  anchors.margins: 6
                  text: ""
                  echoMode: TextInput.Password
                  color: "#ffffff"
                  font.pixelSize: 11
                  verticalAlignment: TextInput.AlignVCenter
                }
              }
            }

            Text {
              visible: errorMessage !== ""
              text: errorMessage
              color: "#ef4444"
              font.pixelSize: 11
              wrapMode: Text.WordWrap
              Layout.fillWidth: true
            }

            Item { Layout.fillHeight: true }

            Rectangle {
              Layout.fillWidth: true
              implicitHeight: 34
              radius: 4
              color: saveMouse.containsMouse ? "#059669" : "#10b981"

              Text {
                anchors.centerIn: parent
                text: "Save Token & Connect"
                color: "#ffffff"
                font.bold: true
                font.pixelSize: 11
              }

              MouseArea {
                id: saveMouse
                anchors.fill: parent
                hoverEnabled: true
                cursorShape: Qt.PointingHandCursor
                onClicked: saveSettings()
              }
            }
          }

          // ── 2. NOTIFICATIONS LIST MODE ──────────────────────────────────
          ColumnLayout {
            visible: !isSetupMode
            Layout.fillWidth: true
            Layout.fillHeight: true
            spacing: 8

            Text {
              visible: isLoading
              text: "Loading notifications..."
              color: "#888888"
              font.pixelSize: 11
              Layout.alignment: Qt.AlignHCenter
            }

            Text {
              visible: !isLoading && errorMessage !== ""
              text: errorMessage
              color: "#ef4444"
              font.pixelSize: 11
              wrapMode: Text.WordWrap
              Layout.fillWidth: true
            }

            Text {
              visible: !isLoading && errorMessage === "" && notificationsList.length === 0
              text: "No notifications right now."
              color: "#666666"
              font.pixelSize: 11
              font.italic: true
              Layout.alignment: Qt.AlignHCenter
            }

            ListView {
              visible: !isLoading && notificationsList.length > 0
              Layout.fillWidth: true
              Layout.fillHeight: true
              clip: true
              spacing: 6
              model: notificationsList

              delegate: Rectangle {
                width: ListView.view.width
                implicitHeight: itemCol.implicitHeight + 16
                radius: 6
                color: "#141414"
                border.color: "#222222"

                ColumnLayout {
                  id: itemCol
                  anchors.left: parent.left
                  anchors.right: parent.right
                  anchors.top: parent.top
                  anchors.margins: 8
                  spacing: 3

                  RowLayout {
                    Layout.fillWidth: true

                    Text {
                      text: modelData.title || "Notification"
                      font.bold: true
                      font.pixelSize: 11
                      color: "#ffffff"
                      Layout.fillWidth: true
                      elide: Text.ElideRight
                    }

                    Rectangle {
                      implicitWidth: 18
                      implicitHeight: 18
                      radius: 4
                      color: markMouse.containsMouse ? "#262626" : "transparent"

                      Text {
                        anchors.centerIn: parent
                        text: "✓"
                        color: "#10b981"
                        font.pixelSize: 11
                      }

                      MouseArea {
                        id: markMouse
                        anchors.fill: parent
                        hoverEnabled: true
                        cursorShape: Qt.PointingHandCursor
                        onClicked: handleMarkRead(modelData.id)
                      }
                    }
                  }

                  Text {
                    text: modelData.message || ""
                    color: "#a0a0a0"
                    font.pixelSize: 10
                    wrapMode: Text.WordWrap
                    Layout.fillWidth: true
                  }
                }
              }
            }

            Rectangle {
              visible: !isLoading && notificationsList.length > 0
              Layout.fillWidth: true
              implicitHeight: 30
              radius: 4
              color: markAllMouse.containsMouse ? "#222222" : "#141414"
              border.color: "#262626"

              Text {
                anchors.centerIn: parent
                text: "Mark All as Read"
                color: "#10b981"
                font.pixelSize: 11
                font.bold: true
              }

              MouseArea {
                id: markAllMouse
                anchors.fill: parent
                hoverEnabled: true
                cursorShape: Qt.PointingHandCursor
                onClicked: handleMarkAllRead()
              }
            }
          }
        }
      }
    }
  }
}
