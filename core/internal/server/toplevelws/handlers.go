package toplevelws

import (
	"fmt"

	"github.com/duxlabs-dev/dux-go/ipc"
	"github.com/duxlabs-dev/dux-shell/core/internal/server/models"
)

func HandleRequest(conn *ipc.ConnWriter, req ipc.Request, m *Manager) {
	if m == nil {
		conn.RespondError(req.ID, "toplevelws manager not initialized")
		return
	}
	var err error
	switch req.Method {
	case "toplevelws.supported":
		conn.Respond(req.ID, map[string]bool{"supported": m.Available()})
		return
	case "toplevelws.move":
		appID, _ := req.Get[string]("appId")
		title, _ := req.Get[string]("title")
		index, _ := req.Get[float64]("index")
		workspace, _ := req.Get[string]("workspace")
		err = m.Move(appID, title, int(index), workspace)
	case "toplevelws.moveDropped":
		id, _ := req.Get[float64]("id")
		workspace, _ := req.Get[string]("workspace")
		err = m.MoveDropped(uint64(id), workspace)
	default:
		err = fmt.Errorf("unknown method: %s", req.Method)
	}
	if err != nil {
		conn.RespondError(req.ID, err.Error())
		return
	}
	conn.Respond(req.ID, models.SuccessResult{Success: true})
}
