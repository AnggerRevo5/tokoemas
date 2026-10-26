package controllers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"

	"toko-emas/models"
	"toko-emas/utils"
)

func ListLokasi(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := db.Query("SELECT id, name, type FROM locations")
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer rows.Close()

		result := []models.Location{}
		for rows.Next() {
			var l models.Location
			if err := rows.Scan(&l.ID, &l.Name, &l.Type); err != nil {
				utils.Gagal(c, err)
				return
			}
			result = append(result, l)
		}
		c.JSON(http.StatusOK, result)
	}
}
