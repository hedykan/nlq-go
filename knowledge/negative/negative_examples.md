# 需要避免的错误模式

本文档包含用户确认不符合预期的查询示例，用于避免常见错误。

---

## 示例
**问题**: 统计每个部门员工数量
**错误SQL**: SELECT shop_name, COUNT(*) as employee_count FROM boom_user GROUP BY shop_name
**说明**: 结果不准确
**正确的SQL**: SELECT department, COUNT(*) FROM boom_user GROUP BY department
---

## 示例
**问题**: SELECT id FROM boom_user JOIN boom_user_sub ON boom_user.shop_id = boom_user_sub.shop_id WHERE is_delete = 0 LIMIT 1
**错误的SQL**: SELECT id FROM boom_user JOIN boom_user_sub ON boom_user.shop_id = boom_user_sub.shop_id WHERE is_delete = 0 LIMIT 1
**错误信息**: 执行SQL失败: Error 1052 (23000): Column 'id' in field list is ambiguous
---

## 示例
**问题**: SELECT * FROM nonexistent_table
**错误的SQL**: SELECT * FROM nonexistent_table
**错误信息**: 执行SQL失败: Error 1146 (42S02): Table 'loloyal.nonexistent_table' doesn't exist
---

## 示例
**问题**: 我现在想查询point_rule中的type为2且is_delete为0的值，然后根据shop_id连表查询他们在boom_user中的shop_name,shopify_plan这个是boom_user表中的plan_displan_name，vip打开状态是查询boom_user中的vip json中的status，0为关1为开，加上app_plan,这个字段使用shop_id查询boom_transaction根据shop_id和payment_status=2查询plan_id之后去boom_plan获取名字，然后还在boom user setting表中根据shop_id和setting_name=points_status 查询setting_value, 值为0和1，总共需要的字段是id,shop_id,data,vip_tier_id,app_plan,shopify_plan,vip_enable,point_rule_status,point_setting_status，注意boom_user_sub的简称为usub和boom_user_setting的简称uset不要搞混,连表查询根据boom_point_rule的id去重,数量应该与select count(*) from boom_point_rule where type = 2 and is_delete = 0 一致
**错误的SQL**: SELECT DISTINCT
**错误信息**: 执行SQL失败: Error 1064 (42000): You have an error in your SQL syntax; check the manual that corresponds to your MySQL server version for the right syntax to use near '' at line 1
---

## 示例
**问题**: 我想知道最近卸载率为啥上升，应该看哪些数据，帮我找回来
**错误的SQL**:
```sql
SELECT
bu.id,
bu.shop_id,
bu.level,
bu.status,
bu.is_delete,
COUNT(bu.id) AS uninstall_count
FROM
boom_user bu
WHERE
bu.level = 'C'
AND bu.status = 0
AND bu.is_delete = 0
GROUP BY
bu.shop_id, bu.level, bu.status, bu.is_delete
ORDER BY
uninstall_count DESC
LIMIT 10;
```
**错误信息**: 执行SQL失败: Error 1055 (42000): Expression #1 of SELECT list is not in GROUP BY clause and contains nonaggregated column 'loloyal.bu.id' which is not functionally dependent on columns in GROUP BY clause; this is incompatible with sql_mode=only_full_group_by
---

## 示例
**问题**: 找出所有下过订单的用户姓名和手机号
**错误的SQL**:
```sql
SELECT DISTINCT CONCAT_WS(' ', first_name, last_name) AS name, phone
FROM boom_customer
INNER JOIN boom_order_paid_water ON boom_customer.customer_rid = boom_order_paid_water.customer_id AND boom_customer.shop_id = boom_order_paid_water.shop_id
WHERE boom_customer.is_delete = 0 AND boom_order_paid_water.is_delete = 0;
```
**错误信息**: 执行SQL失败: Error 1054 (42S22): Unknown column 'boom_customer.shop_id' in 'on clause'
---

## 示例
**问题**: 查询VIP用户的等级、积分和最近一次订单时间
**错误的SQL**:
```sql
SELECT
c.vip_tier_id AS level,
c.points,
MAX(o.created_at) AS last_order_time
FROM
boom_customer c
LEFT JOIN
boom_order_paid_water o ON c.customer_rid = o.customer_id AND c.shop_id = o.shop_id
WHERE
c.vip_tier_id IS NOT NULL
AND c.is_delete = 0
GROUP BY
c.customer_rid,
c.shop_id,
c.vip_tier_id,
c.points;
```
**错误信息**: 执行SQL失败: Error 1054 (42S22): Unknown column 'c.shop_id' in 'on clause'
---

## 示例
**问题**: 查询用户及其关联的客户信息
**错误的SQL**:
```sql
SELECT
u.id,
u.shop_id,
u.shop_name,
u.username,
u.email,
u.level,
u.status,
c.id AS customer_id,
c.customer_rid,
c.first_name,
c.last_name,
c.email AS customer_email,
c.phone,
c.state,
c.total_spent,
c.points
FROM
boom_user u
LEFT JOIN
boom_customer c ON u.shop_id = c.shop_id AND c.is_delete = 0
WHERE
u.is_delete = 0
```
**错误信息**: 执行SQL失败: Error 1054 (42S22): Unknown column 'c.shop_id' in 'on clause'
---

## 示例
**问题**: 查询每个用户的订单数量
**错误的SQL**:
```sql
SELECT
c.customer_rid,
COUNT(o.id) AS order_count
FROM
boom_customer c
LEFT JOIN
boom_order_paid_water o ON c.customer_rid = o.customer_id
WHERE
c.is_delete = 0 AND (o.is_delete = 0 OR o.is_delete IS NULL)
GROUP BY
c.customer_rid;
```
**错误信息**: 执行SQL失败: invalid connection
---

## 示例
**问题**: 统计VIP用户（C级别）的数量和他们的总积分
**错误的SQL**:
```sql
SELECT
COUNT(DISTINCT u.id) AS vip_count,
COALESCE(SUM(c.points), 0) AS total_points
FROM boom_user u
LEFT JOIN boom_customer c ON u.shop_id = c.shop_id AND c.is_delete = 0
WHERE u.level = 'C'
AND u.is_delete = 0
```
**错误信息**: 执行SQL失败: Error 1054 (42S22): Unknown column 'c.shop_id' in 'on clause'