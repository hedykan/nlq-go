# 正面查询示例

本文档包含用户确认符合预期的查询示例，用于提高SQL生成准确性。

---

## 示例
**问题**: 列出现在数据库的所有表名称
**SQL**: SELECT table_name FROM information_schema.tables WHERE table_schema = DATABASE()
**说明**: 符合
---

## 示例
**问题**: 我现在想查询point_rule中的type为2且is_delete为0的值，然后根据shop_id连表查询他们在boom_user中的shop_name,shopify_plan这个是boom_user表中的plan_displan_name，vip打开状态是查询boom_user中的vip json中的status，0为关1为开，加上app_plan,这个字段使用shop_id查询boom_transaction根据shop_id和payment_status=2查询plan_id之后去boom_plan获取名字，然后还在boom user setting表中根据shop_id和setting_name=points_status 查询setting_value, 值为0和1，总共需要的字段是id,shop_id,data,vip_tier_id,app_plan,shopify_plan,vip_enable,point_rule_status,point_setting_status，注意boom_user_sub的简称为usub和boom_user_setting的简称uset不要搞混,连表查询根据boom_point_rule的id去重,数量应该与select count(*) from boom_point_rule where type = 2 and is_delete = 0 一致
**SQL**: SELECT DISTINCT
bpr.id,
bpr.shop_id,
bpr.data,
bpr.vip_tier_id,
bp.name AS app_plan,
bu.plan_display_name AS shopify_plan,
JSON_EXTRACT(bu.vip, '$.status') AS vip_enable,
CASE WHEN JSON_EXTRACT(bu.vip, '$.status') = 1 THEN '开' ELSE '关' END AS point_rule_status,
uset.setting_value AS point_setting_status
FROM
boom_point_rule bpr
JOIN
boom_user bu ON bpr.shop_id = bu.shop_id
LEFT JOIN
boom_transaction bt ON bpr.shop_id = bt.shop_id AND bt.payment_status = 2
LEFT JOIN
boom_plan bp ON bt.plan_id = bp.id
LEFT JOIN
boom_user_setting uset ON bpr.shop_id = uset.shop_id AND uset.setting_name = 'points_status'
WHERE
bpr.type = 2 AND bpr.is_delete = 0
ORDER BY
bpr.id;
**说明**: 太nb了，就是这样
---

## 示例
**问题**: 100个最早的用户的创建时间
**SQL**: SELECT created_at FROM boom_user ORDER BY created_at ASC LIMIT 100
**说明**: 可以，继续保持
---

## 示例
**问题**: 查询一下user人数
**SQL**: SELECT COUNT(*) FROM boom_user
**说明**: 不错
