-- Down: remove seeded stock_sucursal records where stock_actual is still 0
-- (i.e., they were never adjusted, so they're the ones we auto-seeded).
-- NOTE: This is conservative — only removes untouched records.
DELETE FROM stock_sucursal WHERE stock_actual = 0;
