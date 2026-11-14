pragma Singleton
pragma ComponentBehavior: Bound

import QtQuick
import Quickshell
import Quickshell.Wayland
import qs.Common
import qs.Services

// Moves windows to workspaces by dragging (henn labwc fork only): app icons
// dropped on a workspace button, and windows dragged by their titlebar and
// dropped on one. The DMS server talks zhenn_toplevel_workspace_manager_v1.
Singleton {
    id: root
    readonly property var log: Log.scoped("ToplevelWorkspaceService")

    property bool supported: false

    // A titlebar drag ended over a panel: {id, x, y, appId, title}, with
    // x/y in global (layout) coordinates. A workspace button under x/y
    // answers with moveDropped().
    signal toplevelDropped(var data)

    function refresh() {
        if (!DMSService.isConnected) {
            supported = false;
            return;
        }
        DMSService.sendRequest("toplevelws.supported", null, response => {
            root.supported = !!response?.result?.supported;
        });
    }

    // Position among windows with the same app id and title, in creation
    // order, which is how the server tells such windows apart
    function _indexOf(toplevel) {
        const all = ToplevelManager.toplevels?.values ?? [];
        let index = 0;
        for (const t of all) {
            if (t === toplevel)
                return index;
            if (t.appId === toplevel.appId && t.title === toplevel.title)
                index++;
        }
        return -1;
    }

    function moveToplevel(toplevel, workspaceName) {
        if (!supported || !toplevel || !workspaceName)
            return;
        const index = _indexOf(toplevel);
        if (index < 0)
            return;
        DMSService.sendRequest("toplevelws.move", {
            "appId": toplevel.appId ?? "",
            "title": toplevel.title ?? "",
            "index": index,
            "workspace": workspaceName
        }, response => {
            if (response?.error)
                root.log.warn("move failed:", response.error);
        });
    }

    function moveDropped(dropId, workspaceName) {
        DMSService.sendRequest("toplevelws.moveDropped", {
            "id": dropId,
            "workspace": workspaceName
        }, response => {
            if (response?.error)
                root.log.warn("move of dropped window failed:", response.error);
        });
    }

    Component.onCompleted: refresh()

    Connections {
        target: DMSService
        function onIsConnectedChanged() {
            root.refresh();
        }
        function onToplevelWorkspaceDropped(data) {
            root.toplevelDropped(data);
        }
    }
}
