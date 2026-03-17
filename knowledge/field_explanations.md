# 字段说明

## boom_user 表

### 核心字段
- `id`: 用户唯一标识
- `username`: 用户名
- `email`: 邮箱地址
- `phone`: 电话号码
- `level`: 用户等级（C=VIP客户, B=普通客户, A=新客户）
- `status`: 用户状态（1=活跃, 0=非活跃）
- `created_at`: 创建时间（Unix时间戳）
- `updated_at`: 更新时间（Unix时间戳）

### 门店信息
- `shop_id`: 门店ID
- `shop_name`: 门店名称
- `domain`: 门店域名

### 积分相关
- `points_cancel`: 取消的积分
- `reward_channel`: 奖励渠道
- `points_expire_status`: 积分过期状态
- `points_expire_time`: 积分过期时间

## 时间戳字段

所有时间戳字段都是Unix时间戳（秒），需要转换才能直接阅读：
- created_at: 创建时间
- updated_at: 更新时间
- star_feedback_at: 星级评价时间

## 常用查询模式

### 查询VIP用户
```sql
SELECT * FROM boom_user WHERE level = 'C' AND status = 1 AND is_delete = 0
```

### 查询最近创建的用户
```sql
SELECT * FROM boom_user ORDER BY created_at DESC LIMIT 10
```

### 查询活跃用户
```sql
SELECT * FROM boom_user WHERE status = 1 AND is_delete = 0
```
