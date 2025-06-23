package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const baseURL = "http://localhost:8080"

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         struct {
		ID        string    `json:"id"`
		Email     string    `json:"email"`
		Role      string    `json:"role"`
		CreatedAt time.Time `json:"created_at"`
	} `json:"user"`
}

func main() {
	fmt.Println("Testing Watch Party API...")

	// Test health endpoint
	fmt.Println("\n1. Testing Health Check...")
	if err := testHealthCheck(); err != nil {
		fmt.Printf("Health check failed: %v\n", err)
		return
	}
	fmt.Println("✓ Health check passed")

	// Test admin registration
	fmt.Println("\n2. Testing Admin Registration...")
	adminEmail := fmt.Sprintf("admin-%d@example.com", time.Now().Unix())
	if err := testAdminRegistration(adminEmail); err != nil {
		fmt.Printf("Admin registration failed: %v\n", err)
		return
	}
	fmt.Println("✓ Admin registration passed")

	// Test user registration
	fmt.Println("\n3. Testing User Registration...")
	userEmail := fmt.Sprintf("user-%d@example.com", time.Now().Unix())
	if err := testUserRegistration(userEmail); err != nil {
		fmt.Printf("User registration failed: %v\n", err)
		return
	}
	fmt.Println("✓ User registration passed")

	// Test login
	fmt.Println("\n4. Testing User Login...")
	tokens, err := testLogin(userEmail)
	if err != nil {
		fmt.Printf("Login failed: %v\n", err)
		return
	}
	fmt.Println("✓ Login passed")

	// Test logout
	fmt.Println("\n5. Testing User Logout...")
	if err := testLogout(tokens.RefreshToken); err != nil {
		fmt.Printf("Logout failed: %v\n", err)
		return
	}
	fmt.Println("✓ Logout passed")

	fmt.Println("\n🎉 All tests passed!")
}

func testHealthCheck() error {
	resp, err := http.Get(baseURL + "/health")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	return nil
}

func testAdminRegistration(email string) error {
	req := RegisterRequest{
		Email:    email,
		Password: "adminpassword123",
	}

	return testRegistration("/api/v1/admin/register", req)
}

func testUserRegistration(email string) error {
	req := RegisterRequest{
		Email:    email,
		Password: "userpassword123",
	}

	return testRegistration("/api/v1/users/register", req)
}

func testRegistration(endpoint string, req RegisterRequest) error {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return err
	}

	resp, err := http.Post(baseURL+endpoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("expected status 201, got %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func testLogin(email string) (*LoginResponse, error) {
	req := LoginRequest{
		Email:    email,
		Password: "userpassword123",
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(baseURL+"/api/v1/auth/login", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	var loginResp LoginResponse
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &loginResp)
	if err != nil {
		return nil, err
	}

	return &loginResp, nil
}

func testLogout(refreshToken string) error {
	logoutReq := map[string]string{
		"refresh_token": refreshToken,
	}

	jsonData, err := json.Marshal(logoutReq)
	if err != nil {
		return err
	}

	resp, err := http.Post(baseURL+"/api/v1/auth/logout", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
