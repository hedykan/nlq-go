# 业务规则

## VIP用户定义

- VIP用户：level字段为"C"的用户
- VIP用户享受特殊折扣和优惠

## 用户状态

- status = 1：活跃用户
- status = 0：非活跃用户
- is_delete = 1：已删除用户（应被排除在查询之外）

## 折扣规则

- VIP用户（level = "C"）享受20%折扣
- 普通用户无折扣
- 折扣字段：discount_prefix_setting

## 积分规则

- points_expire_status = 1：积分已过期
- points_expire_time：积分过期时间（天数）
- reward_channel：奖励渠道
