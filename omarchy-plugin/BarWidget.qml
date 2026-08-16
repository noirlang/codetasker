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

  function doRefresh() {
    var sv = Api.getSetting("serverUrl", "https://codetasker.noirlang.tr")
    var tk = Api.getSetting("appToken", "")
    if (!tk) { unreadCount = 0; return }
    Api.fetchUnreadCount(sv, tk, function(err, count) {
      if (!err) unreadCount = count
    })
  }

  Component.onCompleted: doRefresh()
  Timer { interval: 10000; repeat: true; running: true; onTriggered: root.doRefresh() }

  // ── Panel lifecycle forwarding ──────────────────────────────────────────
  readonly property bool opened: panelLoader.item ? panelLoader.item.opened === true : false
  function open()   { if (panelLoader.item) panelLoader.item.open() }
  function close()  { if (panelLoader.item) panelLoader.item.close() }
  function togglePanel() { if (panelLoader.item) panelLoader.item.toggle() }
  readonly property bool popoutSwitchClosing: panelLoader.item ? panelLoader.item.popoutSwitchClosing === true : false
  function closeForPopoutSwitch() { if (panelLoader.item) panelLoader.item.closeForPopoutSwitch() }

  function injectPanel() {
    var t = panelLoader.item
    if (!t) return
    if ("bar"        in t) t.bar        = root.bar
    if ("settings"   in t) t.settings   = root.settings
    if ("anchorItem" in t) t.anchorItem = button
    if ("hostWidget" in t) t.hostWidget = root
  }

  implicitWidth:  button.implicitWidth
  implicitHeight: button.implicitHeight

  onBarChanged:      injectPanel()
  onSettingsChanged: injectPanel()

  Loader {
    id: panelLoader
    active: true
    source: Qt.resolvedUrl("Panel.qml")
    visible: false
    onLoaded: { root.injectPanel(); Qt.callLater(root.injectPanel) }
  }

  IpcHandler {
    target: root.moduleName
    function open(): void   { root.open() }
    function close(): void  { root.close() }
    function show(): void   { root.open() }
    function hide(): void   { root.close() }
    function toggle(): void { root.togglePanel() }
    function refresh(): void { root.doRefresh() }
  }

  WidgetButton {
    id: button
    anchors.fill: parent
    bar: root.bar
    text: "\uf121"
    active: root.unreadCount > 0
    useActiveColor: true

    onPressed: function(b) {
      if (b === Qt.RightButton || b === Qt.MiddleButton) root.doRefresh()
      else root.togglePanel()
    }
  }
}
