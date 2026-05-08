-- name: ListMatchingCarServices :many
WITH criteria AS (
    SELECT
        unnest(CAST(sqlc.arg('damage_type_codes') AS text[])) AS damage_type_code,
        unnest(CAST(sqlc.arg('part_category_codes') AS text[])) AS part_category_code
),
matched_profiles AS (
    SELECT
        p.id,
        p.user_id,
        p.organization_name,
        p.city,
        p.address,
        p.phone,
        p.email,
        p.website_url,
        p.contact_info,
        p.description,
        p.is_active,
        p.created_at,
        p.updated_at,
        COUNT(DISTINCT c.damage_type_code || '|' || c.part_category_code)::int AS match_count
    FROM car_service_profiles p
    JOIN car_service_specializations s ON s.profile_id = p.id
    JOIN criteria c ON c.damage_type_code = s.damage_type_code
       AND c.part_category_code = s.part_category_code
    WHERE p.is_active = TRUE
    GROUP BY p.id
)
SELECT
    mp.id,
    mp.user_id,
    mp.organization_name,
    mp.city,
    mp.address,
    mp.phone,
    mp.email,
    mp.website_url,
    mp.contact_info,
    mp.description,
    mp.is_active,
    mp.created_at,
    mp.updated_at,
    mp.match_count
FROM matched_profiles mp
ORDER BY mp.match_count DESC, mp.organization_name ASC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');
