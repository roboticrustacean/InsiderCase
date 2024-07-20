CREATE TABLE teams (
    id INTEGER PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(255) NOT NULL,
    points INTEGER DEFAULT 0,
    won INTEGER DEFAULT 0,
    drawn INTEGER DEFAULT 0,
    lost INTEGER DEFAULT 0,
    goals_scored INTEGER DEFAULT 0,
    goals_conceded INTEGER DEFAULT 0,
    goal_difference INTEGER DEFAULT 0,
    strength INTEGER NOT NULL
);

SELECT * from teams;
SELECT * FROM matches;
DROP TABLE teams;
DROP TABLE matches;

CREATE TABLE matches (
    id INTEGER PRIMARY KEY AUTO_INCREMENT,
    week INTEGER DEFAULT 0,
    home_team_id INTEGER,
    away_team_id INTEGER,
    home_teamsgoals INTEGER,
    away_goals INTEGER,
    FOREIGN KEY (home_team_id) REFERENCES teams(id),
    FOREIGN KEY (away_team_id) REFERENCES teams(id)
);
