<?php
// Include our simple Redis client
require_once 'SimpleRedis.php';

// Set JSON content type
header('Content-Type: application/json');

// Initialize Redis connection
$redis = new SimpleRedis();
try {
    $redis->connect();
} catch (Exception $e) {
    http_response_code(500);
    echo json_encode([
        'error' => true,
        'message' => 'Redis connection failed: ' . $e->getMessage()
    ]);
    exit;
}

// Determine request method
$method = $_SERVER['REQUEST_METHOD'];
$path = parse_url($_SERVER['REQUEST_URI'], PHP_URL_PATH);
$segments = explode('/', trim($path, '/'));
$endpoint = end($segments);

// Redis key prefix for this API
$keyPrefix = 'frango_api_';

// Process the request
switch ($method) {
    case 'GET':
        if (isset($_GET['id'])) {
            // Get specific item
            $id = $_GET['id'];
            $key = $keyPrefix . 'item:' . $id;
            
            if ($redis->exists($key)) {
                $data = json_decode($redis->get($key), true);
                echo json_encode([
                    'id' => $id,
                    'data' => $data,
                    'timestamp' => time(),
                    'retrieved_at' => date('Y-m-d H:i:s')
                ]);
            } else {
                http_response_code(404);
                echo json_encode([
                    'error' => true,
                    'message' => 'Item not found'
                ]);
            }
        } else {
            // List all items
            $keys = $redis->keys($keyPrefix . 'item:*');
            $items = [];
            
            if (is_array($keys)) {
                foreach ($keys as $key) {
                    $id = str_replace($keyPrefix . 'item:', '', $key);
                    $data = json_decode($redis->get($key), true);
                    $items[] = [
                        'id' => $id,
                        'data' => $data
                    ];
                }
            }
            
            echo json_encode([
                'items' => $items,
                'count' => count($items),
                'timestamp' => time()
            ]);
        }
        break;
        
    case 'POST':
        // Create new item
        $input = json_decode(file_get_contents('php://input'), true);
        
        if (!$input) {
            http_response_code(400);
            echo json_encode([
                'error' => true,
                'message' => 'Invalid JSON data'
            ]);
            break;
        }
        
        // Generate unique ID or use provided one
        $id = isset($input['id']) ? $input['id'] : uniqid();
        $key = $keyPrefix . 'item:' . $id;
        
        // Store in Redis
        $redis->set($key, json_encode($input));
        
        // Note: SimpleRedis doesn't support Sets, so we'll track IDs differently
        // Store the ID in a list of all IDs
        $listKey = $keyPrefix . 'items';
        $allItems = $redis->get($listKey);
        $itemIds = $allItems ? json_decode($allItems, true) : [];
        if (!in_array($id, $itemIds)) {
            $itemIds[] = $id;
            $redis->set($listKey, json_encode($itemIds));
        }
        
        echo json_encode([
            'success' => true,
            'id' => $id,
            'message' => 'Item created successfully',
            'timestamp' => time()
        ]);
        break;
        
    case 'PUT':
        // Update existing item
        if (!isset($_GET['id'])) {
            http_response_code(400);
            echo json_encode([
                'error' => true,
                'message' => 'ID parameter is required'
            ]);
            break;
        }
        
        $id = $_GET['id'];
        $key = $keyPrefix . 'item:' . $id;
        
        if (!$redis->exists($key)) {
            http_response_code(404);
            echo json_encode([
                'error' => true,
                'message' => 'Item not found'
            ]);
            break;
        }
        
        $input = json_decode(file_get_contents('php://input'), true);
        
        if (!$input) {
            http_response_code(400);
            echo json_encode([
                'error' => true,
                'message' => 'Invalid JSON data'
            ]);
            break;
        }
        
        // Update in Redis
        $redis->set($key, json_encode($input));
        
        echo json_encode([
            'success' => true,
            'id' => $id,
            'message' => 'Item updated successfully',
            'timestamp' => time()
        ]);
        break;
        
    case 'DELETE':
        // Delete item
        if (!isset($_GET['id'])) {
            http_response_code(400);
            echo json_encode([
                'error' => true,
                'message' => 'ID parameter is required'
            ]);
            break;
        }
        
        $id = $_GET['id'];
        $key = $keyPrefix . 'item:' . $id;
        
        if (!$redis->exists($key)) {
            http_response_code(404);
            echo json_encode([
                'error' => true,
                'message' => 'Item not found'
            ]);
            break;
        }
        
        // Delete from Redis
        $redis->del($key);
        
        // Remove from list of IDs
        $listKey = $keyPrefix . 'items';
        $allItems = $redis->get($listKey);
        if ($allItems) {
            $itemIds = json_decode($allItems, true);
            $itemIds = array_diff($itemIds, [$id]);
            $redis->set($listKey, json_encode($itemIds));
        }
        
        echo json_encode([
            'success' => true,
            'message' => 'Item deleted successfully',
            'timestamp' => time()
        ]);
        break;
        
    default:
        http_response_code(405);
        echo json_encode([
            'error' => true,
            'message' => 'Method not allowed'
        ]);
        break;
} 