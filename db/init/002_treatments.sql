CREATE TABLE IF NOT EXISTS treatments (
    id BIGSERIAL PRIMARY KEY,
    carbs INT,
    insulin NUMERIC(6,2),
    meal_type TEXT,
    food_description TEXT,
    treatment_time BIGINT,
    treatment_datetime TIMESTAMPTZ,
    systime TIMESTAMPTZ
);
