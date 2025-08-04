# Security Documentation

## Overview

This document provides information about the security mechanisms implemented in the application, including authentication, authorization, and data protection.

## Authentication

The application supports multiple authentication methods:

1. Local authentication with username/password
2. LDAP authentication
3. Azure AD authentication

After successful authentication through any of these methods, the system uses JSON Web Tokens (JWT) for maintaining the authenticated session.

### JWT Service

The `JWTService` is responsible for generating and validating JWT tokens.

```
// IJWTService defines the interface for JWT services
type IJWTService interface {
    GenerateToken(userID int, username string) (string, error)
    ValidateToken(token string) (*jwt.Token, error)
    GetUserIDFromToken(token string) (int, error)
}
```

### Token Generation

When a user logs in, the system generates a JWT token containing the user's ID and username. This token is signed with a secret key and has an expiration time.

### Token Validation

For protected endpoints, the system validates the JWT token provided in the `Authorization` header. If the token is valid, the request is allowed to proceed; otherwise, it is rejected with a 401 Unauthorized status.

### Azure AD Authentication

The application supports authentication with Azure Active Directory (Azure AD) using the OAuth 2.0 authorization code flow. This allows users to log in with their Microsoft accounts.

#### Azure AD Authentication Flow

1. The client initiates the authentication process by calling the `/auth/azure-ad/init` endpoint.
2. The server generates a state parameter for CSRF protection and returns an authorization URL.
3. The client redirects the user to the authorization URL, where they log in with their Microsoft credentials.
4. After successful authentication, Azure AD redirects back to the client with an authorization code.
5. The client sends the authorization code to the `/auth/azure-ad/callback` endpoint.
6. The server exchanges the authorization code for an access token and retrieves the user information.
7. If the user doesn't exist in the local database, a new user is created.
8. The server generates JWT tokens for the authenticated user and returns them to the client.

#### Configuration

Azure AD authentication requires the following configuration parameters:

- `AZURE_AD_ENABLED`: Set to "true" to enable Azure AD authentication
- `AZURE_AD_TENANT_ID`: The Azure AD tenant ID
- `AZURE_AD_CLIENT_ID`: The client ID of the registered application in Azure AD
- `AZURE_AD_CLIENT_SECRET`: The client secret of the registered application in Azure AD
- `AZURE_AD_REDIRECT_URI`: The redirect URI registered in Azure AD

## Authorization

The application implements role-based access control (RBAC) to ensure that users can only access resources they are authorized to access.

### Middleware

#### Authentication Middleware

The `RequiresLogin` middleware ensures that the user is authenticated before accessing protected endpoints.

```
// RequiresLogin ensures that the user is authenticated
func RequiresLogin(jwtService security.IJWTService, logger *logger.Logger) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Get the Authorization header
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
            c.Abort()
            return
        }

        // Extract the token from the Authorization header
        tokenString := strings.Replace(authHeader, "Bearer ", "", 1)

        // Validate the token
        token, err := jwtService.ValidateToken(tokenString)
        if err != nil || !token.Valid {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
            c.Abort()
            return
        }

        // Get the user ID from the token
        userID, err := jwtService.GetUserIDFromToken(tokenString)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
            c.Abort()
            return
        }

        // Set the user ID in the context
        c.Set("userID", userID)

        // Continue to the next handler
        c.Next()
    }
}
```

#### Role-Based Authorization Middleware

The `RequiresRole` middleware ensures that the user has the required role to access specific endpoints. This implements role-based access control (RBAC) to restrict access based on user roles.

```
// RequiresRoleMiddleware creates a middleware that checks if the user has the required role
func RequiresRoleMiddleware(requiredRole string, loggerInstance *logger.Logger) gin.HandlerFunc {
    return func(c *gin.Context) {
        tokenString := c.GetHeader("Authorization")
        if tokenString == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Token not provided"})
            c.Abort()
            return
        }

        accessSecret := os.Getenv("JWT_ACCESS_SECRET_KEY")
        if accessSecret == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "JWT_ACCESS_SECRET_KEY not configured"})
            c.Abort()
            return
        }

        tokenString = strings.TrimPrefix(tokenString, "Bearer ")
        claims := jwt.MapClaims{}
        _, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
            return []byte(accessSecret), nil
        })
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
            c.Abort()
            return
        }

        // Check token expiration
        if exp, ok := claims["exp"].(float64); ok {
            if int64(exp) < jwt.TimeFunc().Unix() {
                c.JSON(http.StatusUnauthorized, gin.H{"error": "Token expired"})
                c.Abort()
                return
            }
        } else {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
            c.Abort()
            return
        }

        // Check token type
        if t, ok := claims["type"].(string); ok {
            if t != "access" {
                c.JSON(http.StatusForbidden, gin.H{"error": "Token type mismatch"})
                c.Abort()
                return
            }
        } else {
            c.JSON(http.StatusForbidden, gin.H{"error": "Missing token type"})
            c.Abort()
            return
        }

        // Get user ID from token
        userID, ok := claims["id"].(float64)
        if !ok {
            c.JSON(http.StatusForbidden, gin.H{"error": "Invalid user ID in token"})
            c.Abort()
            return
        }

        // Get user role from token claims
        userRole, ok := claims["role"].(string)
        if !ok {
            loggerInstance.Error("Role claim missing from token", zap.Float64("userID", userID))
            c.JSON(http.StatusForbidden, gin.H{"error": "Invalid token: missing role claim"})
            c.Abort()
            return
        }

        // Check if user has the required role
        if userRole != requiredRole && !(requiredRole == "member" && userRole == "admin") {
            // Admin can do everything a member can do, but not vice versa
            loggerInstance.Warn("User does not have required role", 
                zap.String("requiredRole", requiredRole), 
                zap.String("userRole", userRole),
                zap.Float64("userID", userID))
            c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
            c.Abort()
            return
        }

        // Store user ID and role in context for later use
        c.Set("userID", int(userID))
        c.Set("userRole", userRole)
        c.Next()
    }
}
```

### Role Hierarchy

The application implements a simple role hierarchy:

1. **admin**: Administrators have the highest level of access and can perform all operations.
2. **member**: Regular users with standard permissions.

The role hierarchy is enforced in the `RequiresRole` middleware, which includes a special case where admin users can access endpoints that require the "member" role, but not vice versa. This follows the principle of least privilege while allowing administrators to perform all operations in the system.

### JWT Token Claims

The JWT tokens used for authentication and authorization contain the following claims:

- `id`: The user's ID
- `role`: The user's role (e.g., "admin", "member")
- `type`: The token type (e.g., "access", "refresh")
- `exp`: The token expiration time

These claims are used by the authentication and authorization middlewares to verify the user's identity and permissions.

## Password Security

User passwords are securely hashed using bcrypt before being stored in the database. Bcrypt is a password-hashing function designed to be slow and computationally expensive, making it resistant to brute-force attacks.

```
// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    return string(bytes), err
}

// CheckPasswordHash compares a password with a hash
func CheckPasswordHash(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}
```

## HTTPS

The application should be deployed behind a TLS termination proxy (such as Nginx or a cloud load balancer) to ensure that all communication between clients and the server is encrypted using HTTPS.

## Security Headers

The `Headers` middleware sets security headers to protect against common web vulnerabilities:

```
// Headers sets HTTP headers for security and other purposes
func Headers() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Set security headers
        c.Header("X-Content-Type-Options", "nosniff")
        c.Header("X-Frame-Options", "DENY")
        c.Header("X-XSS-Protection", "1; mode=block")
        c.Header("Content-Security-Policy", "default-src 'self'")
        c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
        c.Header("Feature-Policy", "camera 'none'; microphone 'none'")

        // Set CORS headers
        c.Header("Access-Control-Allow-Origin", "*")
        c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

        // Continue to the next handler
        c.Next()
    }
}
```

## Error Handling

The application implements centralized error handling to ensure that sensitive information is not leaked in error messages. The `ErrorHandler` middleware catches errors and returns appropriate HTTP responses.

## Rate Limiting

To protect against denial-of-service attacks, the application implements rate limiting on sensitive endpoints, such as login and registration.

## Input Validation

All user input is validated using the `validator` package to prevent injection attacks and ensure data integrity.

## Logging

The application logs security-related events, such as login attempts, authentication failures, and authorization violations, to help detect and investigate security incidents.

## Conclusion

Security is a critical aspect of the application, and multiple layers of protection are implemented to ensure the confidentiality, integrity, and availability of the system and its data. Regular security audits and updates are recommended to maintain a strong security posture.
