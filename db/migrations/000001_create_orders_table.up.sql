CREATE TABLE orders (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  status TEXT NOT NULL
    CHECK (status IN (
      'pending',
      'confirmed',
      'preparing',
      'ready',
      'delivered',
      'cancelled'
    )),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  customer_name TEXT,
  notes TEXT,
  total_amount NUMERIC(12, 2)
);

CREATE INDEX idx_orders_status ON orders (status);
CREATE INDEX idx_orders_created_at ON orders (created_at);
