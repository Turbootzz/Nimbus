package handlers

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/services"
)

type BoardHandler struct {
	boards *services.BoardService
}

func NewBoardHandler(boards *services.BoardService) *BoardHandler {
	return &BoardHandler{boards: boards}
}

// A board the user may not see answers like one that doesn't exist
var boardNotFound = map[error]string{repository.ErrBoardNotFound: "Board not found"}

// requestBoard resolves the board a read is about: ?board=<id>, or the
// user's default board. It is the one place a request gets to see a board.
func requestBoard(c *fiber.Ctx, boards *services.BoardService, userID string) (*services.BoardAccess, error) {
	return boards.Resolve(c.Context(), userID, c.Query("board"))
}

// ListBoards returns the user's boards and the boards shared with everyone
func (h *BoardHandler) ListBoards(c *fiber.Ctx) error {
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}
	boards, err := h.boards.List(c.Context(), userID)
	if err != nil {
		return serviceError(c, err, "list boards", boardNotFound)
	}
	response := make([]models.BoardResponse, 0, len(boards))
	for _, board := range boards {
		response = append(response, board.ToResponse(userID))
	}
	return c.JSON(response)
}

// CreateBoard adds a board after the user's last one
func (h *BoardHandler) CreateBoard(c *fiber.Ctx) error {
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}
	var req models.BoardRequest
	if err := c.BodyParser(&req); err != nil {
		return BadRequest(c, "Invalid request body")
	}
	board, err := h.boards.Create(c.Context(), userID, IsAdmin(c), &req)
	if err != nil {
		return serviceError(c, err, "create board", boardNotFound)
	}
	return Created(c, board.ToResponse(userID))
}

// UpdateBoard renames, re-slugs, shares or unshares the user's board
func (h *BoardHandler) UpdateBoard(c *fiber.Ctx) error {
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}
	boardID, err := RequireUUIDParam(c, "id")
	if err != nil {
		return err
	}
	var req models.BoardRequest
	if err := c.BodyParser(&req); err != nil {
		return BadRequest(c, "Invalid request body")
	}
	board, err := h.boards.Update(c.Context(), userID, IsAdmin(c), boardID, &req)
	if err != nil {
		return serviceError(c, err, "update board", boardNotFound)
	}
	return c.JSON(board.ToResponse(userID))
}

// ReorderBoards saves the order of the user's own boards
func (h *BoardHandler) ReorderBoards(c *fiber.Ctx) error {
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}
	var req models.BoardReorderRequest
	if err := c.BodyParser(&req); err != nil {
		return BadRequest(c, "Invalid request body")
	}
	if err := h.boards.Reorder(c.Context(), userID, req.Boards); err != nil {
		return serviceError(c, err, "reorder boards", boardNotFound)
	}
	return c.JSON(fiber.Map{"message": "Board order updated successfully"})
}

// DeleteBoard removes the user's board. Query param move_to=<board id>
// keeps its tiles and groups on that board; without it they are deleted.
func (h *BoardHandler) DeleteBoard(c *fiber.Ctx) error {
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}
	boardID, err := RequireUUIDParam(c, "id")
	if err != nil {
		return err
	}
	if err := h.boards.Delete(c.Context(), userID, strings.ToLower(boardID), c.Query("move_to")); err != nil {
		return serviceError(c, err, "delete board", boardNotFound)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ListBoardIntegrations tells a viewer which kind of integration the
// board's service tiles show, so their numbers get labels. It never returns
// a name, URL or credential.
func (h *BoardHandler) ListBoardIntegrations(c *fiber.Ctx) error {
	userID, err := RequireUserID(c)
	if err != nil {
		return err
	}
	access, err := h.boards.Resolve(c.Context(), userID, c.Params("id"))
	if err != nil {
		return serviceError(c, err, "get board", boardNotFound)
	}
	list, err := h.boards.Integrations(c.Context(), access.Board.ID)
	if err != nil {
		return serviceError(c, err, "list board integrations", boardNotFound)
	}
	return c.JSON(list)
}
