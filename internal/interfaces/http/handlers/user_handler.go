package handlers

import (
	"fmt"
	"strconv"

	"github.com/PavaniTiago/beta-intelligence/internal/application/usecases"
	"github.com/PavaniTiago/beta-intelligence/internal/domain/entities"
	"github.com/PavaniTiago/beta-intelligence/internal/domain/repositories"

	"github.com/gofiber/fiber/v2"
)

type UserHandler struct {
	userUseCase *usecases.UserUseCase
	userRepo    *repositories.UserRepository
}

func NewUserHandler(userUseCase *usecases.UserUseCase, userRepo *repositories.UserRepository) *UserHandler {
	return &UserHandler{
		userUseCase: userUseCase,
		userRepo:    userRepo,
	}
}

func (h *UserHandler) GetUsers(c *fiber.Ctx) error {
	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil || page < 1 {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid 'page' parameter"})
	}

	limit, err := strconv.Atoi(c.Query("limit", "10"))
	if err != nil || limit < 1 {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid 'limit' parameter"})
	}

	sortBy := c.Query("sortBy", "created_at")
	sortDirection := c.Query("sortDirection", "desc")
	orderBy := fmt.Sprintf("%s %s", sortBy, sortDirection)

	users, total, err := h.userUseCase.GetUsers(c.Context(), page, limit, orderBy)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to retrieve users"})
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	return c.JSON(fiber.Map{
		"users":         users,
		"page":          page,
		"limit":         limit,
		"total":         total,
		"totalPages":    totalPages,
		"sortBy":        sortBy,
		"sortDirection": sortDirection,
	})
}

func (h *UserHandler) GetLeads(c *fiber.Ctx) error {
	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil || page < 1 {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid 'page' parameter"})
	}

	limit, err := strconv.Atoi(c.Query("limit", "10"))
	if err != nil || limit < 1 {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid 'limit' parameter"})
	}

	sortBy := c.Query("sortBy", "created_at")
	sortDirection := c.Query("sortDirection", "desc")
	orderBy := fmt.Sprintf("%s %s", sortBy, sortDirection)

	leads, total, err := h.userRepo.FindLeads(page, limit, orderBy)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Erro ao buscar leads: %v", err),
		})
	}

	if leads == nil {
		leads = []entities.User{}
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	return c.JSON(fiber.Map{
		"leads":         leads,
		"page":          page,
		"limit":         limit,
		"total":         total,
		"totalPages":    totalPages,
		"sortBy":        sortBy,
		"sortDirection": sortDirection,
	})
}

func (h *UserHandler) GetClients(c *fiber.Ctx) error {
	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil || page < 1 {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid 'page' parameter"})
	}

	limit, err := strconv.Atoi(c.Query("limit", "10"))
	if err != nil || limit < 1 {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid 'limit' parameter"})
	}

	sortBy := c.Query("sortBy", "created_at")
	sortDirection := c.Query("sortDirection", "desc")
	orderBy := fmt.Sprintf("%s %s", sortBy, sortDirection)

	clients, total, err := h.userRepo.FindClients(page, limit, orderBy)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Erro ao buscar clientes",
		})
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	return c.JSON(fiber.Map{
		"clients":       clients,
		"page":          page,
		"limit":         limit,
		"total":         total,
		"totalPages":    totalPages,
		"sortBy":        sortBy,
		"sortDirection": sortDirection,
	})
}

func (h *UserHandler) GetAnonymous(c *fiber.Ctx) error {
	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil || page < 1 {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid 'page' parameter"})
	}

	limit, err := strconv.Atoi(c.Query("limit", "10"))
	if err != nil || limit < 1 {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid 'limit' parameter"})
	}

	sortBy := c.Query("sortBy", "created_at")
	sortDirection := c.Query("sortDirection", "desc")
	orderBy := fmt.Sprintf("%s %s", sortBy, sortDirection)

	anonymous, total, err := h.userRepo.FindAnonymous(page, limit, orderBy)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Erro ao buscar usuários anônimos",
		})
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	return c.JSON(fiber.Map{
		"anonymous":     anonymous,
		"page":          page,
		"limit":         limit,
		"total":         total,
		"totalPages":    totalPages,
		"sortBy":        sortBy,
		"sortDirection": sortDirection,
	})
}
