import QtQuick
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

  // ── Bar-injected properties ────────────────────────────────────────────
  readonly property color foreground: bar ? bar.barForeground : Color.foreground
  readonly property string fontFamily: bar ? bar.fontFamily : Style.font.family

  // ── State ──────────────────────────────────────────────────────────────
  property int unreadCount: 0
  property string serverUrl: "https://codetasker.noirlang.tr"
  property string appToken: ""
  property bool isSetupMode: false
  property var notificationsList: []
  property bool isLoading: false
  property string errorMessage: ""

  // ── Widget implicit size — must follow the button ──────────────────────
  implicitWidth: button.implicitWidth
  implicitHeight: button.implicitHeight

  // ── IPC routing ───────────────────────────────────────────────────────
  IpcHandler {
    target: root.ipcTarget
    function open(): void  { root.open() }
    function close(): void { root.close() }
    function show(): void  { root.open() }
    function hide(): void  { root.close() }
    function toggle(): void { root.toggle() }
    function refresh(): void { root.doRefresh() }
  }

  // ── Refresh ────────────────────────────────────────────────────────────
  function doRefresh() {
    serverUrl = Api.getSetting("serverUrl", "https://codetasker.noirlang.tr")
    appToken  = Api.getSetting("appToken",  "")

    if (!appToken) {
      unreadCount = 0
      return
    }

    Api.fetchUnreadCount(serverUrl, appToken, function(err, count) {
      if (!err) unreadCount = count
    })
  }

  function loadNotifications() {
    if (!appToken) return
    isLoading = true
    errorMessage = ""
    Api.fetchNotifications(serverUrl, appToken, function(err, items) {
      isLoading = false
      if (err) {
        errorMessage = err
      } else {
        notificationsList = items || []
        unreadCount = notificationsList.length
      }
    })
  }

  function saveSettings() {
    var sv = serverInput.text.trim() || "https://codetasker.noirlang.tr"
    var tk = tokenInput.text.trim()
    Api.setSetting("serverUrl", sv)
    Api.setSetting("appToken",  tk)
    serverUrl = sv
    appToken  = tk
    if (tk) {
      isSetupMode  = false
      errorMessage = ""
      doRefresh()
      loadNotifications()
    } else {
      errorMessage = "Please enter a valid App Token."
    }
  }

  function handleMarkRead(notifId) {
    Api.markRead(serverUrl, appToken, notifId, function() { loadNotifications() })
  }

  function handleMarkAllRead() {
    Api.markAllRead(serverUrl, appToken, function() { loadNotifications() })
  }

  onOpenedChanged: {
    if (opened) {
      doRefresh()
      serverUrl = Api.getSetting("serverUrl", "https://codetasker.noirlang.tr")
      appToken  = Api.getSetting("appToken",  "")
      if (serverInput) serverInput.text = serverUrl
      if (tokenInput)  tokenInput.text  = appToken
      if (!appToken) {
        isSetupMode = true
        Qt.callLater(function() { if (tokenInput) tokenInput.forceActiveFocus() })
      } else {
        isSetupMode = false
        loadNotifications()
        Qt.callLater(function() { keyCatcher.forceActiveFocus() })
      }
    }
  }

  Component.onCompleted: doRefresh()

  Timer {
    interval: 10000
    repeat: true
    running: true
    onTriggered: root.doRefresh()
  }

  // ── Bar button ─────────────────────────────────────────────────────────
  BarIconButton {
    id: button
    anchors.fill: parent
    bar: root.bar
    text: "\uf121"            // fa-code icon — same family as Docker/GitHub icons
    active: root.unreadCount > 0
    activeColor: Color.urgent
    tooltipText: root.unreadCount > 0
      ? "CodeTasker (" + root.unreadCount + " notifications)"
      : "CodeTasker Notifications"

    onPressed: function(b) {
      if (b === Qt.RightButton || b === Qt.MiddleButton) root.doRefresh()
      else root.toggle()
    }
  }

  // ── Popup panel ────────────────────────────────────────────────────────
  KeyboardPanel {
    id: kpanel
    anchorItem: button
    owner: root
    bar: root.bar
    open: root.opened
    focusTarget: keyCatcher
    contentWidth: kpanel.fittedContentWidth(Style.space(380))
    contentHeight: kpanel.fittedContentHeight(content.implicitHeight, Style.space(520))

    PanelKeyCatcher {
      id: keyCatcher
      anchors.fill: parent
      blocked: (serverInput && serverInput.activeFocus) || (tokenInput && tokenInput.activeFocus)
      onCloseRequested: root.close()

      Column {
        id: content
        width: parent.width
        spacing: Style.space(12)

        // Header
        RowLayout {
          width: parent.width
          spacing: Style.space(8)

          Text {
            text: "\uf121"
            color: "#10b981"
            font.pixelSize: Style.font.heading
            font.family: root.fontFamily
          }

          Text {
            text: "CodeTasker"
            color: root.foreground
            font.bold: true
            font.pixelSize: Style.font.heading
            font.family: root.fontFamily
            Layout.fillWidth: true
          }

          // Unread badge
          Rectangle {
            visible: !isSetupMode && unreadCount > 0
            implicitWidth: Math.max(18, badgeText.implicitWidth + 8)
            implicitHeight: 18
            radius: 9
            color: Color.urgent
            Text {
              id: badgeText
              anchors.centerIn: parent
              text: root.unreadCount.toString()
              color: "#fff"
              font.pixelSize: 10
              font.bold: true
            }
          }

          // Refresh
          Rectangle {
            visible: !isSetupMode
            implicitWidth: 26; implicitHeight: 26; radius: 4
            color: refreshMouse.containsMouse ? Color.surface : "transparent"
            Text { anchors.centerIn: parent; text: "↻"; color: root.foreground; font.pixelSize: 12 }
            MouseArea { id: refreshMouse; anchors.fill: parent; hoverEnabled: true; cursorShape: Qt.PointingHandCursor; onClicked: loadNotifications() }
          }

          // Settings gear
          Rectangle {
            implicitWidth: 26; implicitHeight: 26; radius: 4
            color: gearMouse.containsMouse ? Color.surface : "transparent"
            Text { anchors.centerIn: parent; text: "⚙"; color: isSetupMode ? "#10b981" : root.foreground; font.pixelSize: 12 }
            MouseArea { id: gearMouse; anchors.fill: parent; hoverEnabled: true; cursorShape: Qt.PointingHandCursor; onClicked: isSetupMode = !isSetupMode }
          }
        }

        Rectangle { width: parent.width; height: 1; color: Color.surface }

        // ── Setup mode ──────────────────────────────────────────────────
        Column {
          visible: isSetupMode
          width: parent.width
          spacing: Style.space(10)

          Text {
            text: "Connect to CodeTasker"
            font.bold: true; font.pixelSize: Style.font.body
            color: root.foreground; font.family: root.fontFamily
          }
          Text {
            text: "Go to Settings → App Tokens in CodeTasker and paste a generated token below."
            wrapMode: Text.WordWrap; width: parent.width
            font.pixelSize: Style.font.caption
            color: Qt.darker(root.foreground, 1.3); font.family: root.fontFamily
          }

          // Server URL input
          Column { width: parent.width; spacing: Style.space(3)
            Text { text: "SERVER URL"; font.pixelSize: 9; font.bold: true; font.family: "Monospace"; color: Qt.darker(root.foreground, 1.5) }
            Rectangle {
              width: parent.width; height: 32; radius: 4
              color: Color.surface
              border.color: serverInput.activeFocus ? "#10b981" : "transparent"
              TextInput {
                id: serverInput
                anchors { fill: parent; margins: 6 }
                text: "https://codetasker.noirlang.tr"
                color: root.foreground; font.pixelSize: 11
                verticalAlignment: TextInput.AlignVCenter
                selectByMouse: true
                onAccepted: saveSettings()
              }
            }
          }

          // Token input
          Column { width: parent.width; spacing: Style.space(3)
            Text { text: "APP TOKEN"; font.pixelSize: 9; font.bold: true; font.family: "Monospace"; color: Qt.darker(root.foreground, 1.5) }
            Rectangle {
              width: parent.width; height: 32; radius: 4
              color: Color.surface
              border.color: tokenInput.activeFocus ? "#10b981" : "transparent"
              TextInput {
                id: tokenInput
                anchors { fill: parent; margins: 6 }
                color: root.foreground; font.pixelSize: 11
                verticalAlignment: TextInput.AlignVCenter
                selectByMouse: true
                onAccepted: saveSettings()
              }
            }
          }

          Text {
            visible: errorMessage !== ""
            text: errorMessage
            color: Color.urgent; font.pixelSize: 11
            wrapMode: Text.WordWrap; width: parent.width
          }

          Rectangle {
            width: parent.width; height: 34; radius: 4
            color: saveMouse.containsMouse ? "#059669" : "#10b981"
            Text { anchors.centerIn: parent; text: "Save & Connect"; color: "#fff"; font.bold: true; font.pixelSize: 11 }
            MouseArea { id: saveMouse; anchors.fill: parent; hoverEnabled: true; cursorShape: Qt.PointingHandCursor; onClicked: saveSettings() }
          }
        }

        // ── Notifications list ──────────────────────────────────────────
        Column {
          visible: !isSetupMode
          width: parent.width
          spacing: Style.space(6)

          Text {
            visible: isLoading
            text: "Loading…"; color: Qt.darker(root.foreground, 1.3)
            font.pixelSize: 11; font.italic: true
            anchors.horizontalCenter: parent.horizontalCenter
          }
          Text {
            visible: !isLoading && errorMessage !== ""
            text: errorMessage; color: Color.urgent
            font.pixelSize: 11; wrapMode: Text.WordWrap; width: parent.width
          }
          Text {
            visible: !isLoading && errorMessage === "" && notificationsList.length === 0
            text: "No notifications right now."
            color: Qt.darker(root.foreground, 1.5); font.pixelSize: 11; font.italic: true
            anchors.horizontalCenter: parent.horizontalCenter
          }

          Repeater {
            model: notificationsList
            delegate: Rectangle {
              width: content.width
              implicitHeight: nCol.implicitHeight + 14
              radius: 6; color: Color.surface

              Column {
                id: nCol
                anchors { left: parent.left; right: parent.right; top: parent.top; margins: 8 }
                spacing: 3

                RowLayout {
                  width: parent.width
                  Text {
                    text: modelData.title || "Notification"
                    font.bold: true; font.pixelSize: 11; color: root.foreground
                    elide: Text.ElideRight; Layout.fillWidth: true
                  }
                  Rectangle {
                    implicitWidth: 18; implicitHeight: 18; radius: 4
                    color: markMouse.containsMouse ? Qt.darker(Color.surface, 1.2) : "transparent"
                    Text { anchors.centerIn: parent; text: "✓"; color: "#10b981"; font.pixelSize: 11 }
                    MouseArea { id: markMouse; anchors.fill: parent; hoverEnabled: true; cursorShape: Qt.PointingHandCursor; onClicked: handleMarkRead(modelData.id) }
                  }
                }
                Text {
                  text: modelData.message || ""
                  color: Qt.darker(root.foreground, 1.3)
                  font.pixelSize: 10; wrapMode: Text.WordWrap
                  Layout.fillWidth: true
                  width: parent.width
                }
              }
            }
          }

          Rectangle {
            visible: !isLoading && notificationsList.length > 0
            width: parent.width; height: 30; radius: 4
            color: markAllMouse.containsMouse ? Color.surface : "transparent"
            border.color: Color.surface
            Text { anchors.centerIn: parent; text: "Mark All as Read"; color: "#10b981"; font.pixelSize: 11; font.bold: true }
            MouseArea { id: markAllMouse; anchors.fill: parent; hoverEnabled: true; cursorShape: Qt.PointingHandCursor; onClicked: handleMarkAllRead() }
          }
        }
      }
    }
  }
}
