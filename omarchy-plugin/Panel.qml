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

  required property Item anchorItem
  property var hostWidget: null
  readonly property color foreground: bar ? bar.barForeground : Color.foreground
  readonly property string fontFamily: bar ? bar.fontFamily : Style.font.family

  property string serverUrl: "https://codetasker.noirlang.tr"
  property string appToken: ""
  property bool isSetupMode: false
  property var notificationsList: []
  property bool isLoading: false
  property string errorMessage: ""

  function loadSettings() {
    serverUrl = Api.getSetting("serverUrl", "https://codetasker.noirlang.tr")
    appToken  = Api.getSetting("appToken",  "")
    if (serverInput) serverInput.text = serverUrl
    if (tokenInput)  tokenInput.text  = appToken
  }

  function saveSettings() {
    var sv = serverInput ? (serverInput.text.trim() || "https://codetasker.noirlang.tr") : serverUrl
    var tk = tokenInput  ? tokenInput.text.trim() : ""
    Api.setSetting("serverUrl", sv)
    Api.setSetting("appToken",  tk)
    serverUrl = sv
    appToken  = tk
    if (tk) {
      isSetupMode  = false
      errorMessage = ""
      if (hostWidget && hostWidget.doRefresh) hostWidget.doRefresh()
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
      if (err) { errorMessage = err }
      else { notificationsList = items || [] }
    })
  }

  function handleMarkRead(notifId) {
    Api.markRead(serverUrl, appToken, notifId, function() { loadNotifications() })
  }

  function handleMarkAllRead() {
    Api.markAllRead(serverUrl, appToken, function() { loadNotifications() })
  }

  onOpenedChanged: {
    if (opened) {
      loadSettings()
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

  KeyboardPanel {
    id: kpanel
    anchorItem: root.anchorItem
    owner: root
    bar: root.bar
    open: root.opened
    focusTarget: keyCatcher
    contentWidth: kpanel.fittedContentWidth(Style.space(360))
    contentHeight: kpanel.fittedContentHeight(panelCol.implicitHeight, Style.space(520))

    PanelKeyCatcher {
      id: keyCatcher
      anchors.fill: parent
      blocked: (serverInput && serverInput.activeFocus) || (tokenInput && tokenInput.activeFocus)
      onCloseRequested: root.close()

      Column {
        id: panelCol
        width: parent.width
        spacing: Style.space(10)

        // ── Header ──────────────────────────────────────────────────────
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
          // Badge
          Rectangle {
            visible: !isSetupMode && notificationsList.length > 0
            implicitWidth: Math.max(18, bTxt.implicitWidth + 8)
            implicitHeight: 18; radius: 9
            color: Color.urgent
            Text { id: bTxt; anchors.centerIn: parent; text: notificationsList.length.toString(); color: "#fff"; font.pixelSize: 10; font.bold: true }
          }
          // Refresh
          Rectangle {
            visible: !isSetupMode
            implicitWidth: 24; implicitHeight: 24; radius: 4
            color: rfMouse.containsMouse ? Color.surface : "transparent"
            Text { anchors.centerIn: parent; text: "↻"; color: root.foreground; font.pixelSize: 12 }
            MouseArea { id: rfMouse; anchors.fill: parent; hoverEnabled: true; cursorShape: Qt.PointingHandCursor; onClicked: loadNotifications() }
          }
          // Gear
          Rectangle {
            implicitWidth: 24; implicitHeight: 24; radius: 4
            color: grMouse.containsMouse ? Color.surface : "transparent"
            Text { anchors.centerIn: parent; text: "⚙"; color: isSetupMode ? "#10b981" : root.foreground; font.pixelSize: 12 }
            MouseArea { id: grMouse; anchors.fill: parent; hoverEnabled: true; cursorShape: Qt.PointingHandCursor; onClicked: isSetupMode = !isSetupMode }
          }
        }

        Rectangle { width: parent.width; height: 1; color: Color.surface; opacity: 0.6 }

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
            text: "Go to Settings → App Tokens in CodeTasker, generate a token and paste it below."
            wrapMode: Text.WordWrap; width: parent.width
            font.pixelSize: Style.font.caption
            color: Qt.darker(root.foreground, 1.35)
          }

          Column { width: parent.width; spacing: Style.space(3)
            Text { text: "SERVER URL"; font.pixelSize: 9; font.bold: true; font.family: "Monospace"; color: Qt.darker(root.foreground, 1.5) }
            Rectangle {
              width: parent.width; height: 30; radius: 4; color: Color.surface
              border.color: serverInput.activeFocus ? "#10b981" : "transparent"
              TextInput {
                id: serverInput
                anchors { fill: parent; margins: 6 }
                color: root.foreground; font.pixelSize: 11
                verticalAlignment: TextInput.AlignVCenter; selectByMouse: true
                onAccepted: saveSettings()
              }
            }
          }

          Column { width: parent.width; spacing: Style.space(3)
            Text { text: "APP TOKEN"; font.pixelSize: 9; font.bold: true; font.family: "Monospace"; color: Qt.darker(root.foreground, 1.5) }
            Rectangle {
              width: parent.width; height: 30; radius: 4; color: Color.surface
              border.color: tokenInput.activeFocus ? "#10b981" : "transparent"
              TextInput {
                id: tokenInput
                anchors { fill: parent; margins: 6 }
                color: root.foreground; font.pixelSize: 11
                verticalAlignment: TextInput.AlignVCenter; selectByMouse: true
                onAccepted: saveSettings()
              }
            }
          }

          Text {
            visible: errorMessage !== ""
            text: errorMessage; color: Color.urgent
            font.pixelSize: 11; wrapMode: Text.WordWrap; width: parent.width
          }

          Rectangle {
            width: parent.width; height: 32; radius: 4
            color: svMouse.containsMouse ? "#059669" : "#10b981"
            Text { anchors.centerIn: parent; text: "Save Token & Connect"; color: "#fff"; font.bold: true; font.pixelSize: 11 }
            MouseArea { id: svMouse; anchors.fill: parent; hoverEnabled: true; cursorShape: Qt.PointingHandCursor; onClicked: saveSettings() }
          }
        }

        // ── Notification list ────────────────────────────────────────────
        Column {
          visible: !isSetupMode
          width: parent.width
          spacing: Style.space(6)

          Text { visible: isLoading; text: "Loading…"; color: Qt.darker(root.foreground, 1.3); font.pixelSize: 11; font.italic: true; anchors.horizontalCenter: parent.horizontalCenter }
          Text { visible: !isLoading && errorMessage !== ""; text: errorMessage; color: Color.urgent; font.pixelSize: 11; wrapMode: Text.WordWrap; width: parent.width }
          Text { visible: !isLoading && errorMessage === "" && notificationsList.length === 0; text: "No notifications right now."; color: Qt.darker(root.foreground, 1.5); font.pixelSize: 11; font.italic: true; anchors.horizontalCenter: parent.horizontalCenter }

          Repeater {
            model: notificationsList
            delegate: Rectangle {
              width: panelCol.width; implicitHeight: nRow.implicitHeight + 14; radius: 6; color: Color.surface
              RowLayout {
                id: nRow
                anchors { left: parent.left; right: parent.right; top: parent.top; margins: 8 }
                spacing: 6
                Column {
                  Layout.fillWidth: true; spacing: 2
                  Text { text: modelData.title || "Notification"; font.bold: true; font.pixelSize: 11; color: root.foreground; elide: Text.ElideRight; width: parent.width }
                  Text { text: modelData.message || ""; color: Qt.darker(root.foreground, 1.3); font.pixelSize: 10; wrapMode: Text.WordWrap; width: parent.width }
                }
                Rectangle {
                  implicitWidth: 20; implicitHeight: 20; radius: 4
                  color: mkMouse.containsMouse ? Qt.darker(Color.surface, 1.2) : "transparent"
                  Text { anchors.centerIn: parent; text: "✓"; color: "#10b981"; font.pixelSize: 11 }
                  MouseArea { id: mkMouse; anchors.fill: parent; hoverEnabled: true; cursorShape: Qt.PointingHandCursor; onClicked: handleMarkRead(modelData.id) }
                }
              }
            }
          }

          Rectangle {
            visible: !isLoading && notificationsList.length > 0
            width: parent.width; height: 28; radius: 4
            color: maMouse.containsMouse ? Color.surface : "transparent"
            border.color: Color.surface
            Text { anchors.centerIn: parent; text: "Mark All as Read"; color: "#10b981"; font.pixelSize: 11; font.bold: true }
            MouseArea { id: maMouse; anchors.fill: parent; hoverEnabled: true; cursorShape: Qt.PointingHandCursor; onClicked: handleMarkAllRead() }
          }
        }
      }
    }
  }
}
