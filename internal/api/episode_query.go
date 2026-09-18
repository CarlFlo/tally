package api

const episodeSelect = `SELECT e.*,COALESCE((SELECT episode_count FROM seasons WHERE show_id=e.show_id AND number=e.season),0) AS season_episode_count,s.name AS show_name,s.image AS show_image,s.network,f.favorite,COALESCE(p.watched,0) AS watched FROM episodes e JOIN shows s ON s.id=e.show_id JOIN profile_shows f ON f.show_id=s.id AND f.profile_id=? LEFT JOIN profile_episode_state p ON p.episode_id=e.id AND p.profile_id=f.profile_id `
