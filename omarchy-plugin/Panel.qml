import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
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

  readonly property color foreground: bar ? bar.foreground : Color.foreground
  readonly property string fontFamily: bar ? bar.fontFamily : Style.font.family

  property int unreadCount: 0
  property string serverUrl: "https://codetasker.noirlang.tr"
  property string appToken: ""
  property bool isSetupMode: false
  property var notificationsList: []
  property bool isLoading: false
  property string errorMessage: ""

  implicitWidth: button.implicitWidth
  implicitHeight: button.implicitHeight

  function refresh() {
    serverUrl = Api.getSetting("serverUrl", "https://codetasker.noirlang.tr")
    appToken = Api.getSetting("appToken", "")

    if (!appToken) {
      root.unreadCount = 0
      return
    }

    Api.fetchUnreadCount(serverUrl, appToken, function(err, count) {
      if (!err) {
        root.unreadCount = count
      }
    })
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
        refresh()
      }
    })
  }

  function loadSettings() {
    serverUrl = Api.getSetting("serverUrl", "https://codetasker.noirlang.tr")
    appToken = Api.getSetting("appToken", "")
    if (serverInput) serverInput.text = serverUrl
    if (tokenInput) tokenInput.text = appToken
  }

  function saveSettings() {
    serverUrl = serverInput ? (serverInput.text.trim() || "https://codetasker.noirlang.tr") : "https://codetasker.noirlang.tr"
    appToken = tokenInput ? tokenInput.text.trim() : ""

    Api.setSetting("serverUrl", serverUrl)
    Api.setSetting("appToken", appToken)

    if (appToken) {
      isSetupMode = false
      errorMessage = ""
      refresh()
      loadNotifications()
    } else {
      errorMessage = "Please enter a valid App Token."
    }
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

  Component.onCompleted: refresh()

  Timer {
    interval: 10000 // Refresh unread count every 10 seconds
    repeat: true
    running: true
    onTriggered: refresh()
  }

  onOpenedChanged: {
    if (opened) {
      loadSettings()
      if (!appToken) {
        isSetupMode = true
        Qt.callLater(function() { if (tokenInput) tokenInput.forceActiveFocus() })
      } else {
        loadNotifications()
        Qt.callLater(function() { if (keyCatcher) keyCatcher.forceActiveFocus() })
      }
    }
  }

  IpcHandler {
    target: root.ipcTarget
    function open(): void { root.open() }
    function close(): void { root.close() }
    function show(): void { root.open() }
    function hide(): void { root.close() }
    function toggle(): void { root.toggle() }
    function refresh(): string { root.refresh(); return "ok" }
  }

  // Native Omarchy BarIconButton matching system bar widgets
  BarIconButton {
    id: button
    anchors.fill: parent
    bar: root.bar
    text: "" // FontAwesome / Nerd Font code icon (< />)
    active: root.unreadCount > 0
    activeColor: "#ef4444"
    tooltipText: root.unreadCount > 0 ? ("CodeTasker (" + root.unreadCount + " unread)") : "CodeTasker Notifications"

    onPressed: function(buttonCode) {
      if (buttonCode === Qt.RightButton || buttonCode === Qt.MiddleButton) {
        root.refresh()
      } else {
        root.toggle()
      }
    }
  }

  KeyboardPanel {
    id: panel
    anchorItem: button
    owner: root
    bar: root.bar
    open: root.opened
    focusTarget: keyCatcher
    contentWidth: panel.fittedContentWidth(Style.space(380))
    contentHeight: panel.fittedContentHeight(panelColumn.implicitHeight + Style.space(24), Style.space(520))

    PanelKeyCatcher {
      id: keyCatcher
      anchors.fill: parent
      blocked: (serverInput && serverInput.activeFocus) || (tokenInput && tokenInput.activeFocus)
      onCloseRequested: root.close()

      Flickable {
        anchors.fill: parent
        contentWidth: width
        contentHeight: panelColumn.implicitHeight
        clip: true
        boundsBehavior: Flickable.StopAtBounds

        Column {
          id: panelColumn
          width: parent.width
          spacing: Style.space(12)

          // ── Header ────────────────────────────────────────────────────────
          RowLayout {
            width: parent.width
            spacing: Style.space(8)

            Text {
              text: ""
              font.bold: true
              font.pixelSize: Style.font.heading
              font.family: root.fontFamily
              color: "#10b981"
            }

            Text {
              text: "CodeTasker"
              font.bold: true
              font.pixelSize: Style.font.heading
              color: root.foreground
              font.family: root.fontFamily
              Layout.fillWidth: true
            }

            // Unread count badge
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
              color: refreshMouse.containsMouse ? Color.surface : "transparent"

              Text {
                anchors.centerIn: parent
                text: "↻"
                color: root.foreground
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
              color: settingsMouse.containsMouse ? Color.surface : "transparent"

              Text {
                anchors.centerIn: parent
                text: "⚙"
                color: isSetupMode ? "#10b981" : root.foreground
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
            width: parent.width
            height: 1
            color: Color.surface
          }

          // ── 1. SETUP MODE (Token Input Prompt on 1st run) ─────────────────
          Column {
            visible: isSetupMode
            width: parent.width
            spacing: Style.space(12)

            Text {
              text: "Connect to CodeTasker"
              font.bold: true
              font.pixelSize: Style.font.body
              color: root.foreground
              font.family: root.fontFamily
            }

            Text {
              text: "Generate an App Token in CodeTasker (Settings → App Tokens) to receive notifications in Omarchy."
              wrapMode: Text.WordWrap
              width: parent.width
              font.pixelSize: Style.font.caption
              color: Qt.darker(root.foreground, 1.3)
              font.family: root.fontFamily
            }

            Column {
              width: parent.width
              spacing: Style.space(4)

              Text {
                text: "SERVER URL"
                font.pixelSize: 9
                font.bold: true
                font.family: "Monospace"
                color: Qt.darker(root.foreground, 1.5)
              }

              Rectangle {
                width: parent.width
                height: 32
                radius: 4
                color: Color.surface
                border.color: serverInput.activeFocus ? "#10b981" : "transparent"

                TextInput {
                  id: serverInput
                  anchors.fill: parent
                  anchors.margins: 6
                  text: "https://codetasker.noirlang.tr"
                  color: root.foreground
                  font.pixelSize: 11
                  verticalAlignment: TextInput.AlignVCenter
                  selectByMouse: true
                  onAccepted: saveSettings()
                }
              }
            }

            Column {
              width: parent.width
              spacing: Style.space(4)

              Text {
                text: "APP TOKEN (ct_app_...)"
                font.pixelSize: 9
                font.bold: true
                font.family: "Monospace"
                color: Qt.darker(root.foreground, 1.5)
              }

              Rectangle {
                width: parent.width
                height: 32
                radius: 4
                color: Color.surface
                border.color: tokenInput.activeFocus ? "#10b981" : "transparent"

                TextInput {
                  id: tokenInput
                  anchors.fill: parent
                  anchors.margins: 6
                  text: ""
                  color: root.foreground
                  font.pixelSize: 11
                  verticalAlignment: TextInput.AlignVCenter
                  selectByMouse: true
                  onAccepted: saveSettings()
                }
              }
            }

            Text {
              visible: errorMessage !== ""
              text: errorMessage
              color: "#ef4444"
              font.pixelSize: 11
              wrapMode: Text.WordWrap
              width: parent.width
            }

            Rectangle {
              width: parent.width
              height: 34
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
          Column {
            visible: !isSetupMode
            width: parent.width
            spacing: Style.space(8)

            Text {
              visible: isLoading
              text: "Loading notifications..."
              color: Qt.darker(root.foreground, 1.3)
              font.pixelSize: 11
              anchors.horizontalCenter: parent.horizontalCenter
            }

            Text {
              visible: !isLoading && errorMessage !== ""
              text: errorMessage
              color: "#ef4444"
              font.pixelSize: 11
              wrapMode: Text.WordWrap
              width: parent.width
            }

            Text {
              visible: !isLoading && errorMessage === "" && notificationsList.length === 0
              text: "No notifications right now."
              color: Qt.darker(root.foreground, 1.5)
              font.pixelSize: 11
              font.italic: true
              anchors.horizontalCenter: parent.horizontalCenter
            }

            Repeater {
              model: notificationsList
              delegate: Rectangle {
                width: panelColumn.width
                implicitHeight: itemCol.implicitHeight + 16
                radius: 6
                color: Color.surface

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
                      color: root.foreground
                      Layout.fillWidth: true
                      elide: Text.ElideRight
                    }

                    Rectangle {
                      implicitWidth: 18
                      implicitHeight: 18
                      radius: 4
                      color: markMouse.containsMouse ? Qt.darker(Color.surface, 1.2) : "transparent"

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
                    color: Qt.darker(root.foreground, 1.3)
                    font.pixelSize: 10
                    wrapMode: Text.WordWrap
                    Layout.fillWidth: true
                  }
                }
              }
            }

            Rectangle {
              visible: !isLoading && notificationsList.length > 0
              width: parent.width
              height: 30
              radius: 4
              color: markAllMouse.containsMouse ? Color.surface : "transparent"
              border.color: Color.surface

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
