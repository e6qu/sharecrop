-- Collectibles carry a pixel-art sprite slug so the client can render a
-- hand-crafted icon. Existing rows keep an empty slug (rendered as a neutral
-- placeholder); default collectibles minted from the catalog set it.

alter table collectibles
	add column if not exists art text not null default '';
