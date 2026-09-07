package tools

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/LittleCurry/go_first_ai/internal/db"
	"github.com/LittleCurry/go_first_ai/internal/models"
)

// OrderTool 订单查询工具
type OrderTool struct{}

// NewOrderTool 创建订单工具
func NewOrderTool() *OrderTool {
	return &OrderTool{}
}

// GetRecentOrder 获取用户最近一笔订单
func (t *OrderTool) GetRecentOrder(ctx context.Context, userID string) (*models.Order, error) {
	log.Printf("🔍 查询用户 %s 的最近订单", userID)

	query := `
		SELECT order_id, user_id, status, total_amount, shipping_address, 
		       tracking_number, tracking_company, created_at, updated_at
		FROM orders
		WHERE user_id = ?
		ORDER BY created_at DESC
		LIMIT 1
	`

	var order models.Order
	var trackingNumber sql.NullString
	var trackingCompany sql.NullString
	var shippingAddress sql.NullString

	err := db.DB.QueryRowContext(ctx, query, userID).Scan(
		&order.OrderID,
		&order.UserID,
		&order.Status,
		&order.TotalAmount,
		&shippingAddress,
		&trackingNumber,
		&trackingCompany,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query order failed: %w", err)
	}

	// 处理 NULL 值
	if shippingAddress.Valid {
		order.ShippingAddress = shippingAddress.String
	} else {
		order.ShippingAddress = ""
	}
	if trackingNumber.Valid {
		order.TrackingNumber = trackingNumber.String
	} else {
		order.TrackingNumber = ""
	}
	if trackingCompany.Valid {
		order.TrackingCompany = trackingCompany.String
	} else {
		order.TrackingCompany = ""
	}

	// 获取状态中文名
	order.StatusText = getStatusText(order.Status)

	// 获取订单商品
	items, err := t.getOrderItems(ctx, order.OrderID)
	if err != nil {
		log.Printf("get order items failed: %v", err)
	}
	order.Items = items

	// 获取物流轨迹
	logistics, err := t.getLogistics(ctx, order.OrderID)
	if err != nil {
		log.Printf("get logistics failed: %v", err)
	}
	order.Logistics = logistics

	return &order, nil
}

// GetOrderByID 根据订单号查询
func (t *OrderTool) GetOrderByID(ctx context.Context, userID, orderID string) (*models.Order, error) {
	log.Printf("🔍 查询用户 %s 的订单 %s", userID, orderID)

	query := `
		SELECT order_id, user_id, status, total_amount, shipping_address, 
		       tracking_number, tracking_company, created_at, updated_at
		FROM orders
		WHERE user_id = ? AND order_id = ?
	`

	var order models.Order
	var trackingNumber sql.NullString
	var trackingCompany sql.NullString
	var shippingAddress sql.NullString

	err := db.DB.QueryRowContext(ctx, query, userID, orderID).Scan(
		&order.OrderID,
		&order.UserID,
		&order.Status,
		&order.TotalAmount,
		&shippingAddress,
		&trackingNumber,
		&trackingCompany,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query order failed: %w", err)
	}

	// 处理 NULL 值
	if shippingAddress.Valid {
		order.ShippingAddress = shippingAddress.String
	} else {
		order.ShippingAddress = ""
	}
	if trackingNumber.Valid {
		order.TrackingNumber = trackingNumber.String
	} else {
		order.TrackingNumber = ""
	}
	if trackingCompany.Valid {
		order.TrackingCompany = trackingCompany.String
	} else {
		order.TrackingCompany = ""
	}

	order.StatusText = getStatusText(order.Status)

	items, err := t.getOrderItems(ctx, order.OrderID)
	if err != nil {
		log.Printf("get order items failed: %v", err)
	}
	order.Items = items

	logistics, err := t.getLogistics(ctx, order.OrderID)
	if err != nil {
		log.Printf("get logistics failed: %v", err)
	}
	order.Logistics = logistics

	return &order, nil
}

// getOrderItems 获取订单商品
func (t *OrderTool) getOrderItems(ctx context.Context, orderID string) ([]models.OrderItem, error) {
	query := `
		SELECT product_name, product_image, quantity, price
		FROM order_items
		WHERE order_id = ?
	`

	rows, err := db.DB.QueryContext(ctx, query, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.OrderItem
	for rows.Next() {
		var item models.OrderItem
		var productImage sql.NullString
		err := rows.Scan(&item.ProductName, &productImage, &item.Quantity, &item.Price)
		if err != nil {
			continue
		}
		if productImage.Valid {
			item.ProductImage = productImage.String
		} else {
			item.ProductImage = ""
		}
		item.Subtotal = item.Price * float64(item.Quantity)
		items = append(items, item)
	}
	return items, nil
}

// getLogistics 获取物流轨迹
func (t *OrderTool) getLogistics(ctx context.Context, orderID string) ([]models.Logistics, error) {
	query := `
		SELECT status, location, description, tracking_time
		FROM logistics_tracking
		WHERE order_id = ?
		ORDER BY tracking_time DESC
	`

	rows, err := db.DB.QueryContext(ctx, query, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logistics []models.Logistics
	for rows.Next() {
		var item models.Logistics
		var location sql.NullString
		var description sql.NullString
		err := rows.Scan(&item.Status, &location, &description, &item.Time)
		if err != nil {
			continue
		}
		if location.Valid {
			item.Location = location.String
		} else {
			item.Location = ""
		}
		if description.Valid {
			item.Description = description.String
		} else {
			item.Description = ""
		}
		item.StatusText = getLogisticsStatusText(item.Status)
		logistics = append(logistics, item)
	}
	return logistics, nil
}

// GetUserCoupons 获取用户可用优惠券
func (t *OrderTool) GetUserCoupons(ctx context.Context, userID string) ([]models.Coupon, error) {
	query := `
		SELECT coupon_code, name, discount_type, discount_value, min_amount, status, expire_at
		FROM coupons
		WHERE user_id = ? AND status = 'available' AND expire_at > NOW()
		ORDER BY expire_at ASC
	`

	rows, err := db.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var coupons []models.Coupon
	for rows.Next() {
		var c models.Coupon
		var name sql.NullString
		err := rows.Scan(&c.CouponCode, &name, &c.DiscountType, &c.DiscountValue, &c.MinAmount, &c.Status, &c.ExpireAt)
		if err != nil {
			continue
		}
		if name.Valid {
			c.Name = name.String
		} else {
			c.Name = ""
		}
		coupons = append(coupons, c)
	}
	return coupons, nil
}

// getStatusText 获取订单状态中文
func getStatusText(status string) string {
	switch status {
	case "pending":
		return "待付款"
	case "paid":
		return "待发货"
	case "shipped":
		return "已发货"
	case "delivered":
		return "已送达"
	case "cancelled":
		return "已取消"
	default:
		return status
	}
}

// getLogisticsStatusText 获取物流状态中文
func getLogisticsStatusText(status string) string {
	switch status {
	case "pending":
		return "待发货"
	case "picked":
		return "已揽收"
	case "in_transit":
		return "运输中"
	case "delivered":
		return "已签收"
	default:
		return status
	}
}

// FormatOrderForDisplay 格式化订单信息用于显示
func FormatOrderForDisplay(order *models.Order) string {
	if order == nil || order.OrderID == "" {
		return "暂无订单信息"
	}

	statusEmoji := map[string]string{
		"pending":   "⏳",
		"paid":      "📦",
		"shipped":   "🚚",
		"delivered": "✅",
		"cancelled": "❌",
	}

	emoji := statusEmoji[order.Status]
	if emoji == "" {
		emoji = "📋"
	}

	result := fmt.Sprintf(`%s 订单号：%s
状态：%s
金额：%.2f 元
下单时间：%s
`,
		emoji,
		order.OrderID,
		order.StatusText,
		order.TotalAmount,
		order.CreatedAt.Format("2006-01-02 15:04"),
	)

	if order.TrackingNumber != "" {
		result += fmt.Sprintf("物流：%s %s\n", order.TrackingCompany, order.TrackingNumber)
	}

	if len(order.Items) > 0 {
		result += "\n商品清单：\n"
		for _, item := range order.Items {
			result += fmt.Sprintf("  • %s x%d (%.2f元/件)\n",
				item.ProductName,
				item.Quantity,
				item.Price,
			)
		}
	}

	if len(order.Logistics) > 0 {
		result += "\n最新物流状态：\n"
		latest := order.Logistics[0]
		result += fmt.Sprintf("  %s %s\n", latest.StatusText, latest.Description)
	}

	return result
}

// ProcessRefund 处理退款申请
func (t *OrderTool) ProcessRefund(ctx context.Context, userID, orderID, reason string) (*models.RefundResult, error) {
	log.Printf("💰 处理退款申请: user=%s, order=%s, reason=%s", userID, orderID, reason)

	// 1. 查询订单信息
	order, err := t.GetOrderByID(ctx, userID, orderID)
	if err != nil {
		return nil, fmt.Errorf("查询订单失败: %w", err)
	}
	if order == nil {
		return nil, fmt.Errorf("订单不存在")
	}

	// 2. 检查订单状态是否允许退款
	allowedStatus := map[string]bool{
		"paid":    true,
		"shipped": true,
	}
	if !allowedStatus[order.Status] {
		return &models.RefundResult{
			Success: false,
			Message: fmt.Sprintf("当前订单状态为【%s】，不支持申请退款。如需帮助，请联系人工客服。", order.StatusText),
			OrderID: orderID,
			Status:  "failed",
		}, nil
	}

	// 3. 生成退款单号
	refundID := fmt.Sprintf("REF-%s-%d", time.Now().Format("20060102"), time.Now().UnixNano()%10000)

	// 4. 更新订单状态为"退款中"
	updateQuery := `UPDATE orders SET status = 'refunding' WHERE order_id = ?`
	_, err = db.DB.ExecContext(ctx, updateQuery, orderID)
	if err != nil {
		return nil, fmt.Errorf("更新订单状态失败: %w", err)
	}

	// 5. 创建退款记录（这里简化，实际应该插入退款表）
	// 实际项目中需要创建 refunds 表

	result := &models.RefundResult{
		Success:      true,
		Message:      fmt.Sprintf("✅ 退款申请已提交！\n退款单号：%s\n订单号：%s\n退款金额：%.2f元\n预计1-3个工作日到账。", refundID, orderID, order.TotalAmount),
		RefundID:     refundID,
		OrderID:      orderID,
		Status:       "pending",
		RefundAmount: order.TotalAmount,
	}

	log.Printf("✅ 退款申请成功: refundID=%s", refundID)
	return result, nil
}

// GetOrderLogistics 获取订单物流轨迹（完整版）
func (t *OrderTool) GetOrderLogistics(ctx context.Context, orderID string) ([]models.Logistics, error) {
	query := `
		SELECT status, location, description, tracking_time
		FROM logistics_tracking
		WHERE order_id = ?
		ORDER BY tracking_time DESC
	`

	rows, err := db.DB.QueryContext(ctx, query, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logistics []models.Logistics
	for rows.Next() {
		var item models.Logistics
		var location sql.NullString
		var description sql.NullString
		err := rows.Scan(&item.Status, &location, &description, &item.Time)
		if err != nil {
			continue
		}
		if location.Valid {
			item.Location = location.String
		}
		if description.Valid {
			item.Description = description.String
		}
		item.StatusText = getLogisticsStatusText(item.Status)
		logistics = append(logistics, item)
	}
	return logistics, nil
}

// RepurchaseOrder 再来一单
func (t *OrderTool) RepurchaseOrder(ctx context.Context, userID, orderID string) (*models.Order, error) {
	log.Printf("🔄 再来一单: user=%s, order=%s", userID, orderID)

	// 1. 查询原订单
	order, err := t.GetOrderByID(ctx, userID, orderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, fmt.Errorf("订单不存在")
	}

	// 2. 创建新订单（这里简化，实际需要重新创建订单）
	newOrderID := fmt.Sprintf("ORD-%s-%d", time.Now().Format("20060102"), time.Now().UnixNano()%10000)

	insertQuery := `
		INSERT INTO orders (order_id, user_id, status, total_amount, shipping_address, created_at)
		VALUES (?, ?, 'paid', ?, ?, NOW())
	`
	_, err = db.DB.ExecContext(ctx, insertQuery, newOrderID, userID, order.TotalAmount, order.ShippingAddress)
	if err != nil {
		return nil, fmt.Errorf("创建新订单失败: %w", err)
	}

	// 3. 复制商品
	for _, item := range order.Items {
		itemQuery := `
			INSERT INTO order_items (order_id, product_name, product_image, quantity, price)
			VALUES (?, ?, ?, ?, ?)
		`
		_, err = db.DB.ExecContext(ctx, itemQuery, newOrderID, item.ProductName, item.ProductImage, item.Quantity, item.Price)
		if err != nil {
			log.Printf("复制商品失败: %v", err)
		}
	}

	// 4. 返回新订单
	return t.GetOrderByID(ctx, userID, newOrderID)
}
