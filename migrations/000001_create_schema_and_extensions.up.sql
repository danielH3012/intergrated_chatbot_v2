-- 000001_create_schema_and_extensions.up.sql
-- Enable necessary extensions and auxiliary schemas

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE SCHEMA IF NOT EXISTS schema1;
