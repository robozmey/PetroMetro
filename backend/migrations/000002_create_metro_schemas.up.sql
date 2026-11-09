CREATE TABLE metro_schemas (
    city_id BIGINT PRIMARY KEY REFERENCES cities(id) ON DELETE CASCADE,
    schema JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT metro_schemas_document CHECK (
        jsonb_typeof(schema) = 'object'
        AND schema ? 'stations' AND schema ? 'lines'
        AND jsonb_typeof(schema->'stations') = 'array'
        AND jsonb_typeof(schema->'lines') = 'array'
        AND NOT (schema ? 'city')
    )
);
