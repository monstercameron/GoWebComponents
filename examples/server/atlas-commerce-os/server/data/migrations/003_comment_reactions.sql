alter table comments add column reaction text not null default 'up';

update comments
set reaction = case
    when lower(coalesce(status, '')) in ('flagged', 'rejected') then 'down'
    else 'up'
end
;