CREATE TABLE IF NOT EXISTS readings (
    sensor_id   TEXT        NOT NULL,
    room        TEXT        NOT NULL,
    type        TEXT        NOT NULL,
    value       FLOAT8      NOT NULL,
    unit        TEXT        NOT NULL,
    time        TIMESTAMPTZ NOT NULL
);

SELECT create_hypertable('readings', 'time', if_not_exists => TRUE);
