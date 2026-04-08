create table if not exists purchase_orders (
    id text primary key,
    vendor_name text not null,
    warehouse_id text not null references warehouses(id),
    status text not null,
    priority_note text not null,
    eta text not null,
    created_at text not null,
    updated_at text not null
);

create table if not exists purchase_order_lines (
    id text primary key,
    purchase_order_id text not null references purchase_orders(id) on delete cascade,
    product_sku text not null references products(sku),
    quantity integer not null,
    eta text not null,
    status text not null
);

create table if not exists inventory_threshold_events (
    id text primary key,
    product_sku text not null references products(sku) on delete cascade,
    warehouse_id text not null references warehouses(id),
    reorder_point integer not null,
    safety_stock integer not null,
    actor_name text not null,
    summary text not null,
    detail text not null,
    created_at text not null
);

create index if not exists idx_purchase_orders_warehouse_status on purchase_orders(warehouse_id, status);
create index if not exists idx_purchase_order_lines_order on purchase_order_lines(purchase_order_id);
create index if not exists idx_threshold_events_product_created on inventory_threshold_events(product_sku, created_at desc);