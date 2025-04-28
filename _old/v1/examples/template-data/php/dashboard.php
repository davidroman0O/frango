<?php
header('Content-Type: text/html');
// Access the data from the nested structure
$dashboard = $data;
?>
<!DOCTYPE html>
<html>
<head>
    <title><?= htmlspecialchars($dashboard['page_title']) ?></title>
    <style>
        body {
            font-family: system-ui, -apple-system, sans-serif;
            margin: 0;
            padding: 0;
            background: #f5f7f9;
            color: #333;
        }
        .layout {
            display: flex;
            min-height: 100vh;
        }
        .sidebar {
            width: 250px;
            background: #2c3e50;
            color: white;
            padding: 20px 0;
        }
        .content {
            flex: 1;
            padding: 20px;
        }
        .header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 0 0 20px 0;
            border-bottom: 1px solid #e1e4e8;
            margin-bottom: 20px;
        }
        .user-info {
            display: flex;
            align-items: center;
        }
        .avatar {
            width: 40px;
            height: 40px;
            border-radius: 50%;
            background: #0366d6;
            color: white;
            display: flex;
            align-items: center;
            justify-content: center;
            margin-right: 10px;
            font-weight: bold;
        }
        .notifications {
            position: relative;
        }
        .notification-badge {
            position: absolute;
            top: -5px;
            right: -5px;
            background: #e74c3c;
            color: white;
            border-radius: 50%;
            width: 18px;
            height: 18px;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 12px;
        }
        .logo {
            font-size: 24px;
            font-weight: bold;
            padding: 0 20px 20px 20px;
            border-bottom: 1px solid #3d5165;
            margin-bottom: 20px;
        }
        .nav-item {
            padding: 12px 20px;
            display: flex;
            align-items: center;
            cursor: pointer;
        }
        .nav-item:hover {
            background: #3d5165;
        }
        .nav-item.active {
            background: #0366d6;
        }
        .nav-icon {
            margin-right: 10px;
            width: 20px;
            text-align: center;
        }
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
            gap: 20px;
            margin-bottom: 20px;
        }
        .stat-card {
            background: white;
            border-radius: 4px;
            padding: 20px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.1);
        }
        .stat-value {
            font-size: 28px;
            font-weight: bold;
            margin-bottom: 5px;
        }
        .stat-label {
            color: #666;
            font-size: 14px;
        }
        .card {
            background: white;
            border-radius: 4px;
            padding: 20px;
            margin-bottom: 20px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.1);
        }
        .activity-item {
            padding: 12px 0;
            border-bottom: 1px solid #e1e4e8;
            display: flex;
            align-items: center;
        }
        .activity-icon {
            width: 30px;
            height: 30px;
            border-radius: 50%;
            background: #f1f8ff;
            color: #0366d6;
            display: flex;
            align-items: center;
            justify-content: center;
            margin-right: 15px;
        }
        .activity-content {
            flex: 1;
        }
        .activity-time {
            color: #666;
            font-size: 12px;
        }
        h1, h2, h3 {
            margin-top: 0;
        }
        .back-link {
            display: inline-block;
            margin-top: 20px;
            color: #0366d6;
            text-decoration: none;
        }
        .back-link:hover {
            text-decoration: underline;
        }
    </style>
</head>
<body>
    <div class="layout">
        <!-- Sidebar -->
        <div class="sidebar">
            <div class="logo">Frango Admin</div>
            
            <nav>
                <?php foreach ($dashboard['navigation_items'] as $navItem): ?>
                    <div class="nav-item <?= $navItem['name'] === 'Dashboard' ? 'active' : '' ?>">
                        <span class="nav-icon"><?= getIconSymbol($navItem['icon']) ?></span>
                        <?= htmlspecialchars($navItem['name']) ?>
                    </div>
                <?php endforeach; ?>
            </nav>
        </div>
        
        <!-- Main Content -->
        <div class="content">
            <!-- Header -->
            <div class="header">
                <h1><?= htmlspecialchars($dashboard['page_title']) ?></h1>
                
                <div class="user-info">
                    <div class="notifications">
                        <span>🔔</span>
                        <?php if ($dashboard['notification_count'] > 0): ?>
                            <span class="notification-badge"><?= $dashboard['notification_count'] ?></span>
                        <?php endif; ?>
                    </div>
                    
                    <div class="avatar" title="<?= htmlspecialchars($dashboard['user']['name']) ?>">
                        <?= getInitials($dashboard['user']['name']) ?>
                    </div>
                    
                    <div>
                        <div><?= htmlspecialchars($dashboard['user']['name']) ?></div>
                        <div style="font-size: 12px; color: #666;"><?= htmlspecialchars($dashboard['user']['role']) ?></div>
                    </div>
                </div>
            </div>
            
            <!-- Stats Grid -->
            <div class="stats-grid">
                <?php foreach ($dashboard['stats'] as $key => $value): ?>
                    <div class="stat-card">
                        <div class="stat-value"><?= htmlspecialchars($value) ?></div>
                        <div class="stat-label"><?= ucfirst(str_replace('_', ' ', $key)) ?></div>
                    </div>
                <?php endforeach; ?>
            </div>
            
            <!-- Recent Activities -->
            <div class="card">
                <h2>Recent Activities</h2>
                
                <?php foreach ($dashboard['recent_activities'] as $activity): ?>
                    <div class="activity-item">
                        <div class="activity-icon">
                            <?= getActivityIcon($activity['type']) ?>
                        </div>
                        <div class="activity-content">
                            <div><?= htmlspecialchars($activity['message']) ?></div>
                            <div class="activity-time"><?= formatTimestamp($activity['timestamp']) ?></div>
                        </div>
                    </div>
                <?php endforeach; ?>
            </div>
            
            <!-- User Info Card -->
            <div class="card">
                <h2>User Information</h2>
                <div style="display: grid; grid-template-columns: 120px 1fr; gap: 10px;">
                    <div>User ID:</div>
                    <div><?= $dashboard['user']['id'] ?></div>
                    
                    <div>Name:</div>
                    <div><?= htmlspecialchars($dashboard['user']['name']) ?></div>
                    
                    <div>Email:</div>
                    <div><?= htmlspecialchars($dashboard['user']['email']) ?></div>
                    
                    <div>Role:</div>
                    <div><?= htmlspecialchars($dashboard['user']['role']) ?></div>
                    
                    <div>Created At:</div>
                    <div><?= formatDate($dashboard['user']['created_at']) ?></div>
                </div>
            </div>
            
            <a href="/" class="back-link">Back to home</a>
        </div>
    </div>
    
    <?php
    // Helper functions
    
    // Get initials from a name
    function getInitials($name) {
        $words = explode(" ", $name);
        $initials = "";
        
        foreach ($words as $word) {
            $initials .= strtoupper(substr($word, 0, 1));
        }
        
        return substr($initials, 0, 2);
    }
    
    // Format a timestamp for display
    function formatTimestamp($timestamp) {
        $time = strtotime($timestamp);
        $now = time();
        $diff = $now - $time;
        
        if ($diff < 60) {
            return "Just now";
        } elseif ($diff < 3600) {
            $minutes = floor($diff / 60);
            return $minutes . " minute" . ($minutes > 1 ? "s" : "") . " ago";
        } elseif ($diff < 86400) {
            $hours = floor($diff / 3600);
            return $hours . " hour" . ($hours > 1 ? "s" : "") . " ago";
        } else {
            $days = floor($diff / 86400);
            return $days . " day" . ($days > 1 ? "s" : "") . " ago";
        }
    }
    
    // Format a date for display
    function formatDate($date) {
        return date("F j, Y", strtotime($date));
    }
    
    // Get an icon symbol based on the icon name
    function getIconSymbol($iconName) {
        $icons = [
            "dashboard" => "📊",
            "user" => "👤",
            "settings" => "⚙️",
            "logout" => "🚪",
            "default" => "📄"
        ];
        
        return $icons[$iconName] ?? $icons["default"];
    }
    
    // Get an activity icon based on the activity type
    function getActivityIcon($type) {
        $icons = [
            "login" => "🔑",
            "update" => "📝",
            "api" => "🔌",
            "security" => "🔒",
            "default" => "📄"
        ];
        
        return $icons[$type] ?? $icons["default"];
    }
    ?>
</body>
</html> 