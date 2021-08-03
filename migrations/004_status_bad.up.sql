CREATE TABLE service_down (
    status_id SERIAL PRIMARY KEY,
    service text,
    status text,
    timestamp timestamptz
);