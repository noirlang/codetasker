import QtQuick
import Quickshell
import Quickshell.Io
import qs.Commons
import qs.Ui
import "CodeTaskerApi.js" as Api

BarWidget {
  id: root
  moduleName: "codetasker.notifications"

  property int unreadCount: 0
  property string serverUrl: "https://codetasker.noirlang.tr"
  property string appToken: ""

  function refresh() {
    serverUrl = Api.getSetting("serverUrl", "https://codetasker.noirlang.tr")
    appToken = Api.getSetting("appToken", "")

    if (!appToken) {
      root.unreadCount = 0
      if (panelLoader.item && panelLoader.item.refresh) panelLoader.item.refresh()
      return
    }

    Api.fetchUnreadCount(serverUrl, appToken, function(err, count) {
      if (!err) {
        root.unreadCount = count
      }
      if (panelLoader.item && panelLoader.item.refresh) panelLoader.item.refresh()
    })
  }

  Component.onCompleted: root.refresh()

  Timer {
    interval: 10000 // Refresh unread count every 10 seconds
    repeat: true
    running: true
    onTriggered: root.refresh()
  }

  function injectPanel() {
    var target = panelLoader.item
    if (!target) return
    if ("bar" in target) target.bar = root.bar
    if ("settings" in target) target.settings = root.settings
    if ("anchorItem" in target) target.anchorItem = button
    if ("hostWidget" in target) target.hostWidget = root
  }

  // Forward panel lifecycle properties and methods for shell summon / hide / toggle routing
  readonly property bool opened: panelLoader.item ? panelLoader.item.opened === true : false

  function open() {
    if (panelLoader.item && panelLoader.item.open) panelLoader.item.open()
  }

  function close() {
    if (panelLoader.item && panelLoader.item.close) panelLoader.item.close()
  }

  function togglePanel() {
    if (panelLoader.item && panelLoader.item.toggle) panelLoader.item.toggle()
  }

  readonly property bool popoutSwitchClosing: panelLoader.item ? panelLoader.item.popoutSwitchClosing === true : false

  function closeForPopoutSwitch() {
    if (panelLoader.item && panelLoader.item.closeForPopoutSwitch) panelLoader.item.closeForPopoutSwitch()
  }

  implicitWidth: button.implicitWidth
  implicitHeight: button.implicitHeight

  onBarChanged: injectPanel()
  onSettingsChanged: injectPanel()

  Loader {
    id: panelLoader
    active: true
    source: Qt.resolvedUrl("Panel.qml")
    visible: false
    onLoaded: {
      root.injectPanel()
      Qt.callLater(root.injectPanel)
    }
  }

  BarIconButton {
    id: button
    anchors.fill: parent
    bar: root.bar
    text: "󰅩" // Nerd Font md-code_tags (< />) icon matching system bar icons
    active: root.unreadCount > 0
    activeColor: "#ef4444"
    tooltipText: root.unreadCount > 0 ? ("CodeTasker (" + root.unreadCount + " unread)") : "CodeTasker Notifications"

    onPressed: function(b) {
      if (b === Qt.RightButton || b === Qt.MiddleButton) {
        root.refresh()
      } else {
        root.togglePanel()
      }
    }
  }
}
