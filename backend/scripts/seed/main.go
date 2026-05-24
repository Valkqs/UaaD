// Package main provides a large-scale seed script for Sprint 4.
// Creates 1000 users, 100 activities, ~6000 enrollments, ~3600 orders, 5000 behaviors (total >15k rows).
// Usage: from backend/ directory run go run ./scripts/seed
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/joho/godotenv"
	"github.com/uaad/backend/internal/config"
	"github.com/uaad/backend/internal/domain"
	"golang.org/x/crypto/bcrypt"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	totalUsers      = 1000
	totalActivities = 100
	enrollsPerUser  = 6   // each user enrolls in 6 unique activities → ~6000 enrollments
	behaviorTotal   = 5000
)

func main() {
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../.env")

	cfg := config.Load()
	db, err := gorm.Open(gormmysql.Open(cfg.MySQLDSN()), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	cfg.ApplyMySQLPool(sqlDB)

	if err := db.AutoMigrate(
		&domain.User{},
		&domain.Activity{},
		&domain.Enrollment{},
		&domain.Order{},
		&domain.UserBehavior{},
		&domain.Notification{},
		&domain.ActivityScore{},
	); err != nil {
		log.Fatalf("failed to migrate: %v", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("test123456"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("failed to hash password: %v", err)
	}
	hashed := string(hash)

	// ── 1. Fixed test accounts (preserved across seed runs) ──────────────────
	fixedUsers := []domain.User{
		{Phone: "13800000001", Username: "测试用户1", PasswordHash: hashed, Role: "USER"},
		{Phone: "13800000002", Username: "测试用户2", PasswordHash: hashed, Role: "USER"},
		{Phone: "13800000003", Username: "测试用户3", PasswordHash: hashed, Role: "USER"},
		{Phone: "13800000004", Username: "李明商户", PasswordHash: hashed, Role: "MERCHANT"},
		{Phone: "13800000005", Username: "王芳商户", PasswordHash: hashed, Role: "MERCHANT"},
	}
	for i := range fixedUsers {
		db.Where("phone = ?", fixedUsers[i].Phone).FirstOrCreate(&fixedUsers[i])
	}
	merchantIDs := []uint64{fixedUsers[3].ID, fixedUsers[4].ID}
	fmt.Printf("✅ Fixed accounts ready (merchant IDs: %v)\n", merchantIDs)

	// ── 2. Bulk users (phone: 18900000000 ~ 18900000994) ─────────────────────
	fmt.Print("📦 Seeding bulk users...")
	bulkUsers := make([]domain.User, 0, totalUsers-len(fixedUsers))
	for i := 0; i < totalUsers-len(fixedUsers); i++ {
		bulkUsers = append(bulkUsers, domain.User{
			Phone:        fmt.Sprintf("189%08d", i),
			Username:     fmt.Sprintf("用户_%05d", i),
			PasswordHash: hashed,
			Role:         "USER",
		})
	}
	// ON DUPLICATE KEY DO NOTHING — safe to re-run
	db.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(bulkUsers, 200)
	fmt.Printf(" done\n")

	// Load all users to get real IDs
	var allUsers []domain.User
	db.Select("id").Find(&allUsers)
	fmt.Printf("✅ %d users total in DB\n", len(allUsers))

	// ── 3. Activities ─────────────────────────────────────────────────────────
	fmt.Print("📦 Seeding activities...")
	now := time.Now()
	categories := []string{"CONCERT", "CONFERENCE", "EXPO", "ESPORTS", "EXHIBITION", "OTHER"}
	cities := []string{"北京国家体育场", "上海梅赛德斯奔驰文化中心", "广州天河体育场", "深圳湾体育场", "成都凤凰山体育场",
		"杭州奥体中心", "武汉光谷国际会展中心", "西安奥体中心", "南京奥体中心", "重庆奥体中心"}

	for i := 1; i <= totalActivities; i++ {
		cat := categories[i%len(categories)]
		merchantID := merchantIDs[i%len(merchantIDs)]
		location := cities[i%len(cities)]

		// First 20: keep original logic (preserve existing data)
		// 21-40: "hot" PUBLISHED activities with enrollment window OPEN NOW (for JMeter)
		// 41-70: PUBLISHED activities with future enrollment window
		// 71-100: DRAFT/PREHEAT activities
		var status string
		var enrollOpen, enrollClose time.Time
		switch {
		case i <= 20:
			statuses := []string{"PUBLISHED", "PREHEAT", "DRAFT", "PUBLISHED", "PUBLISHED"}
			status = statuses[i%len(statuses)]
			enrollOpen = now.Add(time.Duration(-24+i) * time.Hour)
			enrollClose = now.Add(time.Duration(48+i*2) * time.Hour)
		case i <= 40:
			status = "PUBLISHED"
			enrollOpen = now.Add(-2 * time.Hour)           // opened 2 hours ago
			enrollClose = now.Add(time.Duration(i) * 24 * time.Hour) // closes in i days
		case i <= 70:
			status = "PUBLISHED"
			enrollOpen = now.Add(time.Duration(i-40) * 24 * time.Hour)  // opens in future
			enrollClose = now.Add(time.Duration(i-40+30) * 24 * time.Hour)
		default:
			statuses := []string{"DRAFT", "PREHEAT"}
			status = statuses[i%len(statuses)]
			enrollOpen = now.Add(time.Duration(i) * 24 * time.Hour)
			enrollClose = now.Add(time.Duration(i+30) * 24 * time.Hour)
		}

		tags := fmt.Sprintf("[\"热门\",\"%s\",\"大型活动\"]", cat)
		activity := &domain.Activity{
			Title:         fmt.Sprintf("大型活动 #%03d · %s", i, cat),
			Description:   fmt.Sprintf("这是第 %d 场大型%s活动，地点位于%s。本次活动将汇聚全国顶尖表演者，规模宏大，不容错过。", i, cat, location),
			Location:      location,
			Category:      cat,
			Price:         float64((i%10+1) * 99),
			Status:        status,
			CreatedBy:     merchantID,
			MaxCapacity:   5000 + (i%5)*2000,
			EnrollOpenAt:  enrollOpen,
			EnrollCloseAt: enrollClose,
			ActivityAt:    enrollClose.Add(30 * 24 * time.Hour),
			EnrollCount:   int64(i * 300),
			ViewCount:     int64(i * 1500),
			Tags:          &tags,
		}
		db.Where("title = ?", activity.Title).FirstOrCreate(activity)
	}
	var allActivities []domain.Activity
	db.Select("id, price, status").Find(&allActivities)
	fmt.Printf(" done\n✅ %d activities total in DB\n", len(allActivities))

	if len(allActivities) == 0 {
		log.Fatal("no activities found, aborting")
	}

	// ── 4. Enrollments & Orders ───────────────────────────────────────────────
	// For user at index i, enroll in activities at positions:
	//   (i*13 + k*17) % len(allActivities) for k = 0..enrollsPerUser-1
	// 17 and typical activity counts are coprime → 6 distinct activities per user.
	fmt.Printf("📦 Seeding enrollments and orders (may take 30–60s)...\n")

	numAct := len(allActivities)
	orderCounter := 0
	enrollBatch := make([]domain.Enrollment, 0, 500)
	orderBatch := make([]domain.Order, 0, 500)

	flushEnrollBatch := func() {
		if len(enrollBatch) == 0 {
			return
		}
		db.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(enrollBatch, 200)
		enrollBatch = enrollBatch[:0]
	}
	flushOrderBatch := func() {
		if len(orderBatch) == 0 {
			return
		}
		db.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(orderBatch, 200)
		orderBatch = orderBatch[:0]
	}

	enrolledAt := now.Add(-30 * 24 * time.Hour) // historical: enrolled 30 days ago

	for ui, u := range allUsers {
		for k := 0; k < enrollsPerUser; k++ {
			actIdx := (ui*13 + k*17) % numAct
			act := allActivities[actIdx]

			// Alternate enrollment statuses for variety
			status := "SUCCESS"
			switch (ui + k) % 5 {
			case 3:
				status = "CANCELLED"
			case 4:
				status = "QUEUING"
			}

			queuePos := ui*enrollsPerUser + k + 1
			finalizedAt := enrolledAt.Add(5 * time.Second)

			enroll := domain.Enrollment{
				UserID:        u.ID,
				ActivityID:    act.ID,
				Status:        status,
				QueuePosition: &queuePos,
				EnrolledAt:    enrolledAt,
				FinalizedAt:   &finalizedAt,
			}
			enrollBatch = append(enrollBatch, enroll)

			if len(enrollBatch) >= 500 {
				flushEnrollBatch()
			}
		}

		if (ui+1)%100 == 0 {
			fmt.Printf("  enrollments: %d/%d users processed\n", ui+1, len(allUsers))
		}
	}
	flushEnrollBatch()

	// Load enrollments to generate orders for SUCCESS ones
	var successEnrollments []domain.Enrollment
	db.Where("status = ?", "SUCCESS").Find(&successEnrollments)
	fmt.Printf("  found %d SUCCESS enrollments, generating orders (60%%)...\n", len(successEnrollments))

	// Build activity price map
	priceMap := make(map[uint64]float64, len(allActivities))
	for _, a := range allActivities {
		priceMap[a.ID] = a.Price
	}

	for ei, enroll := range successEnrollments {
		// 60% of SUCCESS enrollments get orders
		if ei%5 >= 3 {
			continue
		}
		orderCounter++

		var orderStatus string
		var paidAt *time.Time
		switch orderCounter % 4 {
		case 0:
			orderStatus = "PAID"
			t := enroll.EnrolledAt.Add(10 * time.Minute)
			paidAt = &t
		case 1:
			orderStatus = "PENDING"
		case 2:
			orderStatus = "CLOSED"
		default:
			orderStatus = "PAID"
			t := enroll.EnrolledAt.Add(5 * time.Minute)
			paidAt = &t
		}

		orderBatch = append(orderBatch, domain.Order{
			OrderNo:      fmt.Sprintf("ORD%016d", orderCounter),
			EnrollmentID: enroll.ID,
			UserID:       enroll.UserID,
			ActivityID:   enroll.ActivityID,
			Amount:       priceMap[enroll.ActivityID],
			Status:       orderStatus,
			PaidAt:       paidAt,
			ExpiredAt:    enroll.EnrolledAt.Add(30 * time.Minute),
			CreatedAt:    enroll.EnrolledAt,
			UpdatedAt:    enroll.EnrolledAt.Add(2 * time.Minute),
		})

		if len(orderBatch) >= 500 {
			flushOrderBatch()
		}
	}
	flushOrderBatch()

	var enrollCount, orderCount int64
	db.Model(&domain.Enrollment{}).Count(&enrollCount)
	db.Model(&domain.Order{}).Count(&orderCount)
	fmt.Printf("✅ %d enrollments, %d orders in DB\n", enrollCount, orderCount)

	// ── 5. User Behaviors ─────────────────────────────────────────────────────
	fmt.Print("📦 Seeding user behaviors...")
	behaviorTypes := []string{"VIEW", "COLLECT", "SHARE", "CLICK"}
	behaviorBatch := make([]domain.UserBehavior, 0, 500)

	for i := 0; i < behaviorTotal; i++ {
		u := allUsers[i%len(allUsers)]
		act := allActivities[(i*7+3)%numAct]
		bType := behaviorTypes[i%len(behaviorTypes)]
		detail := fmt.Sprintf("{\"source\":\"home\",\"seq\":%d}", i)

		behaviorBatch = append(behaviorBatch, domain.UserBehavior{
			UserID:       u.ID,
			ActivityID:   act.ID,
			BehaviorType: bType,
			Detail:       &detail,
			CreatedAt:    now.Add(-time.Duration(i%720) * time.Hour),
		})

		if len(behaviorBatch) >= 500 {
			db.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(behaviorBatch, 200)
			behaviorBatch = behaviorBatch[:0]
		}
	}
	if len(behaviorBatch) > 0 {
		db.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(behaviorBatch, 200)
	}

	var behaviorCount int64
	db.Model(&domain.UserBehavior{}).Count(&behaviorCount)
	fmt.Printf(" done\n✅ %d behaviors in DB\n", behaviorCount)

	// ── Summary ───────────────────────────────────────────────────────────────
	var userCount, actCount int64
	db.Model(&domain.User{}).Count(&userCount)
	db.Model(&domain.Activity{}).Count(&actCount)

	total := userCount + actCount + enrollCount + orderCount + behaviorCount
	fmt.Printf("\n🎉 Seed complete!\n")
	fmt.Printf("   Users:       %d\n", userCount)
	fmt.Printf("   Activities:  %d\n", actCount)
	fmt.Printf("   Enrollments: %d\n", enrollCount)
	fmt.Printf("   Orders:      %d\n", orderCount)
	fmt.Printf("   Behaviors:   %d\n", behaviorCount)
	fmt.Printf("   ─────────────────────\n")
	fmt.Printf("   TOTAL ROWS:  %d\n", total)
	if total >= 10000 {
		fmt.Printf("   ✅ 达到万条数据要求！\n")
	} else {
		fmt.Printf("   ⚠️  未达到万条，当前 %d 条\n", total)
	}
	fmt.Printf("\nRun `go run ./cmd/server` to start the API.\n")
}
