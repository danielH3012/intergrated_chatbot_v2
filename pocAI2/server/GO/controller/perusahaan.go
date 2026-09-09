package controllers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
)

type Perusahaan struct {
	IdPerusahaan   int    `json:"id_perusahaan"`
	NamaPerusahaan string `json:"nama_perusahaan"`
}

func GetPerusahaan(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := DB.Query(ctx, `SELECT id_perusahaan, nama_perusahaan FROM perusahaan ORDER BY id_perusahaan ASC`)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	defer rows.Close()

	var list []Perusahaan
	for rows.Next() {
		var p Perusahaan
		if err := rows.Scan(&p.IdPerusahaan, &p.NamaPerusahaan); err == nil {
			list = append(list, p)
		}
	}

	if list == nil {
		list = []Perusahaan{}
	}

	return c.JSON(list)
}
