-- name: InsertReading :exec
INSERT INTO readings (sensor_id, room, type, value, unit, time)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetReadingsByRoomAndType :many
SELECT sensor_id, room, type, value, unit, time
FROM readings
WHERE room = $1
  AND type = $2
  AND time >= $3
  AND time <= $4
ORDER BY time DESC;
