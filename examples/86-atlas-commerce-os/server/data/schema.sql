-- Atlas Commerce OS initial schema scaffold

create table if not exists warehouses (
    id text primary key,
    slug text not null unique,
    name text not null,
    region text not null,
    service_level text not null,
    public_summary text not null,
    created_at text not null,
    updated_at text not null
);

create table if not exists products (
    sku text primary key,
    slug text not null unique,
    title text not null,
    category text not null,
    price_cents integer not null,
    status text not null,
    finish text not null,
    summary text not null,
    details text not null,
    seo_title text not null,
    seo_description text not null,
    created_at text not null,
    updated_at text not null
);

create table if not exists product_media (
    id text primary key,
    product_sku text not null references products(sku) on delete cascade,
    kind text not null,
    url text not null,
    alt_text text not null,
    position integer not null default 0
);

create table if not exists inventory_levels (
    id text primary key,
    product_sku text not null references products(sku) on delete cascade,
    warehouse_id text not null references warehouses(id) on delete cascade,
    on_hand integer not null,
    reserved integer not null,
    available integer not null,
    inbound integer not null,
    damaged integer not null,
    reorder_point integer not null,
    safety_stock integer not null,
    status text not null,
    updated_at text not null,
    unique(product_sku, warehouse_id)
);

create table if not exists saved_views (
    id text primary key,
    name text not null,
    owner_id text not null,
    scope text not null,
    filters_json text not null,
    sort_key text not null,
    sort_direction text not null,
    density text not null,
    warehouse_id text,
    created_at text not null,
    updated_at text not null
);

create table if not exists comments (
    id text primary key,
    product_sku text not null references products(sku) on delete cascade,
    author_name text not null,
    author_type text not null,
    reaction text not null default 'up',
    subject text not null,
    body text not null,
    status text not null,
    moderation_reason text,
    created_at text not null,
    updated_at text not null
);

create table if not exists quote_requests (
    id text primary key,
    product_sku text not null references products(sku) on delete cascade,
    requester_name text not null,
    company_name text,
    email text not null,
    quantity integer not null,
    note text not null,
    created_at text not null
);

create table if not exists restock_requests (
    id text primary key,
    product_sku text not null references products(sku) on delete cascade,
    email text not null,
    preferred_warehouse_id text references warehouses(id),
    created_at text not null
);

create table if not exists transfers (
    id text primary key,
    source_warehouse_id text not null references warehouses(id),
    destination_warehouse_id text not null references warehouses(id),
    status text not null,
    reason text not null,
    recommended_by text not null,
    created_at text not null,
    updated_at text not null
);

create table if not exists transfer_lines (
    id text primary key,
    transfer_id text not null references transfers(id) on delete cascade,
    product_sku text not null references products(sku),
    quantity integer not null
);

create table if not exists receiving_sessions (
    id text primary key,
    source_type text not null,
    source_id text not null,
    warehouse_id text not null references warehouses(id),
    status text not null,
    discrepancy_summary text,
    created_at text not null,
    updated_at text not null
);

create table if not exists receiving_lines (
    id text primary key,
    receiving_session_id text not null references receiving_sessions(id) on delete cascade,
    product_sku text not null references products(sku),
    expected_quantity integer not null,
    actual_quantity integer not null,
    discrepancy_reason text not null default ''
);

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

create table if not exists preferences (
    id text primary key,
    owner_id text not null unique,
    theme text not null,
    locale text not null,
    density text not null,
    default_warehouse_id text references warehouses(id),
    created_at text not null,
    updated_at text not null
);

create table if not exists audit_events (
    id text primary key,
    actor_id text not null,
    event_type text not null,
    entity_type text not null,
    entity_id text not null,
    payload_json text not null,
    created_at text not null
);

create index if not exists idx_products_category on products(category);
create index if not exists idx_inventory_levels_status on inventory_levels(status);
create index if not exists idx_inventory_levels_warehouse on inventory_levels(warehouse_id);
create index if not exists idx_comments_status on comments(status);
create index if not exists idx_saved_views_owner_scope on saved_views(owner_id, scope);
create index if not exists idx_audit_events_entity on audit_events(entity_type, entity_id);
create index if not exists idx_purchase_orders_warehouse_status on purchase_orders(warehouse_id, status);
create index if not exists idx_purchase_order_lines_order on purchase_order_lines(purchase_order_id);
create index if not exists idx_threshold_events_product_created on inventory_threshold_events(product_sku, created_at desc);