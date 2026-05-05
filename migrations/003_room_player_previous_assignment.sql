ALTER TABLE room_players ADD COLUMN previous_team TEXT NOT NULL DEFAULT '';
ALTER TABLE room_players ADD COLUMN previous_spymaster INTEGER NOT NULL DEFAULT 0;
ALTER TABLE room_players ADD COLUMN previous_representative INTEGER NOT NULL DEFAULT 0;
