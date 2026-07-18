CREATE TABLE IF NOT EXISTS nightscoutdb (
    id BIGSERIAL PRIMARY KEY,
    sgv INT,
    ns_time BIGINT,
    ns_datetime TIMESTAMPTZ,
    trend INT,
    utcoffset INT,
    systime TIMESTAMPTZ
);
