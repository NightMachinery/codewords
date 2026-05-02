ALTER TABLE room_players ADD COLUMN mod INTEGER NOT NULL DEFAULT 0;

UPDATE room_players
SET mod = 1
WHERE EXISTS (
  SELECT 1 FROM rooms WHERE rooms.id = room_players.room_id AND rooms.host_user_id = room_players.user_id
);
