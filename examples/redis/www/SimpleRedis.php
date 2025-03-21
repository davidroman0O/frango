<?php
/**
 * SimpleRedis - A pure PHP Redis client
 * 
 * This class provides basic Redis functionality without requiring the Redis extension
 */
class SimpleRedis {
    private $socket;
    private $host;
    private $port;
    private $timeout;
    private $connected = false;
    
    public function __construct($host = '127.0.0.1', $port = 6379, $timeout = 2.0) {
        $this->host = $host;
        $this->port = $port;
        $this->timeout = $timeout;
    }
    
    public function connect() {
        $this->socket = @fsockopen($this->host, $this->port, $errno, $errstr, $this->timeout);
        if (!$this->socket) {
            throw new Exception("Could not connect to Redis: $errstr ($errno)");
        }
        stream_set_timeout($this->socket, $this->timeout);
        $this->connected = true;
        return true;
    }
    
    public function set($key, $value) {
        if (!$this->connected) $this->connect();
        $command = "*3\r\n$3\r\nSET\r\n$" . strlen($key) . "\r\n$key\r\n$" . strlen($value) . "\r\n$value\r\n";
        fwrite($this->socket, $command);
        return $this->readResponse();
    }
    
    public function get($key) {
        if (!$this->connected) $this->connect();
        $command = "*2\r\n$3\r\nGET\r\n$" . strlen($key) . "\r\n$key\r\n";
        fwrite($this->socket, $command);
        return $this->readResponse();
    }
    
    public function incr($key) {
        if (!$this->connected) $this->connect();
        $command = "*2\r\n$4\r\nINCR\r\n$" . strlen($key) . "\r\n$key\r\n";
        fwrite($this->socket, $command);
        return $this->readResponse();
    }
    
    public function exists($key) {
        if (!$this->connected) $this->connect();
        $command = "*2\r\n$6\r\nEXISTS\r\n$" . strlen($key) . "\r\n$key\r\n";
        fwrite($this->socket, $command);
        return (bool) $this->readResponse();
    }
    
    public function keys($pattern) {
        if (!$this->connected) $this->connect();
        $command = "*2\r\n$4\r\nKEYS\r\n$" . strlen($pattern) . "\r\n$pattern\r\n";
        fwrite($this->socket, $command);
        return $this->readResponse();
    }
    
    public function del($key) {
        if (!$this->connected) $this->connect();
        $command = "*2\r\n$3\r\nDEL\r\n$" . strlen($key) . "\r\n$key\r\n";
        fwrite($this->socket, $command);
        return $this->readResponse();
    }
    
    public function info() {
        if (!$this->connected) $this->connect();
        $command = "*1\r\n$4\r\nINFO\r\n";
        fwrite($this->socket, $command);
        $response = $this->readResponse();
        
        $info = [];
        $lines = explode("\r\n", $response);
        foreach ($lines as $line) {
            if (empty($line) || $line[0] == '#') continue;
            $parts = explode(':', $line, 2);
            if (count($parts) == 2) {
                $info[$parts[0]] = $parts[1];
            }
        }
        return $info;
    }
    
    private function readResponse() {
        $line = fgets($this->socket);
        if ($line === false) {
            throw new Exception("Failed to read from socket");
        }
        
        $type = $line[0];
        $line = substr($line, 1, -2); // Remove type char and \r\n
        
        switch ($type) {
            case '+': // Status reply
                return $line;
            case '-': // Error reply
                throw new Exception("Redis error: " . $line);
            case ':': // Integer reply
                return (int) $line;
            case '$': // Bulk reply
                $length = (int) $line;
                if ($length == -1) return null; // Null bulk reply
                $data = '';
                while ($length > 0) {
                    $chunk = fread($this->socket, $length);
                    $data .= $chunk;
                    $length -= strlen($chunk);
                }
                fread($this->socket, 2); // Discard \r\n
                return $data;
            case '*': // Multi-bulk reply
                $count = (int) $line;
                if ($count == -1) return null;
                $data = [];
                for ($i = 0; $i < $count; $i++) {
                    $data[] = $this->readResponse();
                }
                return $data;
            default:
                throw new Exception("Unknown response type: " . $type);
        }
    }
    
    public function __destruct() {
        if ($this->socket) {
            fclose($this->socket);
        }
    }
} 