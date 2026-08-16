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
      return
    }

    Api.fetchUnreadCount(serverUrl, appToken, function(err, count) {
      if (!err) {
        root.unreadCount = count
      }
    })
  }

  Component.onCompleted: root.refresh()

  Timer {
    interval: 10000 // Refresh every 10 seconds
    repeat: true
    running: true
    onTriggered: root.refresh()
  }

  function injectPanel() {
    var target = panelLoader.item
    if (!target) return
    if ("bar" in target) target.bar = root.bar
    if ("anchorItem" in target) target.anchorItem = button
    if ("hostWidget" in target) target.hostWidget = root
  }

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

  onBarChanged: injectPanel()

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

  WidgetButton {
    id: button
    anchors.fill: parent
    bar: root.bar
    text: "</>" + (root.unreadCount > 0 ? (" (" + root.unreadCount + ")") : "")
    labelVisible: !root.vertical
    hasVisualContent: true
    horizontalMargin: 8.75
    verticalPadding: 8.75

    // Icon color: Red (#ef4444) if unreadCount > 0, White (#ffffff) if unreadCount == 0
    foreground: root.unreadCount > 0 ? "#ef4444" : "#ffffff"

    onPressed: function(b) {
      if (b === Qt.LeftButton) {
        root.refresh()
        root.togglePanel()
      }
    }
  }
}
