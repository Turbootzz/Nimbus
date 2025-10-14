# Database Seeder

A Go-based database seeder for creating test data in the Nimbus application.

## 🔒 Production Safety

This seeder uses **build tags** to ensure it's **completely excluded** from production builds:

```go
// +build dev
```

### What This Means:

✅ **Development**: Seeder is available with `-tags dev` flag  
❌ **Production**: Seeder is completely excluded from binary  
✅ **Safe**: No risk of accidentally running seed data in production  
✅ **Clean**: No dead code in production binary  

This is a **compile-time guarantee** that test data cannot leak into production.

## 📁 File Structure

```text
backend/
├── cmd/seed/
│   ├── main.go              # Entry point (build tag: dev)
│   └── README.md            # This file
└── internal/seeds/
    └── users.go             # User seeding logic (build tag: dev)
```

**Benefits:**
- ✅ Separation of concerns
- ✅ Reusable seed functions  
- ✅ Easy to add more seeders
- ✅ Protected by build tags

## Usage

```bash
make seed    # Uses -tags dev automatically
```

All test users have password: **`password123`**

See full documentation in the file for test scenarios and details.
