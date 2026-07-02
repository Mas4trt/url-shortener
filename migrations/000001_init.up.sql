CREATE SCHEMA IF NOT EXISTS urlshortener;

CREATE TABLE IF NOT EXISTS urlshortener.url(
	id SERIAL PRIMARY KEY,
	alias TEXT NOT NULL UNIQUE,
	url TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_alias ON urlshortener.url(alias);