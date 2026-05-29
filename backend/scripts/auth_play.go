package main

import (
	"fmt"
	"time"

	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/auth"
	"github.com/google/uuid"
)

func main() {
	hash, _ := auth.HashPassword("Password123!")
	fmt.Println("Hash:", hash)
	fmt.Println("Verfiy (correct):", auth.VerifyPassword("Password123!", hash))
	fmt.Println("Verify (wrong):", auth.VerifyPassword("Password123!_wrong", hash))

	// === JWT ====
	jwtSvc := auth.NewJWTservice("a-test-secret-that-is-long-enough!", time.Hour, 24*time.Hour)
	tid, uid := uuid.New(), uuid.New()
	tok, _ := jwtSvc.GenerateAccessToken(tid, uid, "admin", auth.ClientWeb)
	fmt.Println("\nJWT:", tok)
	claims, _ := jwtSvc.ParseAccessToken(tok)
	fmt.Printf("Claims : tid=%s , uid=%s\n", claims.TenantId, claims.UserId)

	// === Refresh Token ===
	raw, refesh_hash, _ := auth.GenerateRefreshToken()
	fmt.Println("\nRefesh raw:", raw)
	fmt.Println("\nRefresh hash:", refesh_hash)

}
