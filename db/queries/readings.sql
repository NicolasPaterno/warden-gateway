-- name: InsertReading :exec
INSERT INTO readings (tenant_id, sensor_id, room, type, value, unit, time)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetReadingsByRoomAndType :many
SELECT tenant_id, sensor_id, room, type, value, unit, time
FROM readings
WHERE tenant_id = $1
  AND room = $2
  AND type = $3
  AND time >= $4
  AND time <= $5
ORDER BY time DESC;
