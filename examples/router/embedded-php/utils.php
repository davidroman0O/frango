<?php
/**
 * Frango Utilities Library
 * 
 * A collection of helper functions for PHP pages in the Frango application.
 */

/**
 * Format a currency value with the appropriate symbol
 * 
 * @param float $amount The amount to format
 * @param string $currency The currency code (default: USD)
 * @return string Formatted currency string
 */
function format_currency($amount, $currency = 'USD') {
    $symbols = [
        'USD' => '$',
        'EUR' => '€',
        'GBP' => '£',
        'JPY' => '¥',
    ];
    
    $symbol = isset($symbols[$currency]) ? $symbols[$currency] : '';
    return $symbol . number_format($amount, 2);
}

/**
 * Get the user's first name from a full name
 * 
 * @param string $fullName The full name
 * @return string The first name
 */
function get_first_name($fullName) {
    $parts = explode(' ', $fullName);
    return $parts[0];
}

/**
 * Format a date in a nice readable format
 * 
 * @param string $date Date string or timestamp
 * @param string $format The date format (default: readable format)
 * @return string Formatted date
 */
function format_date($date, $format = null) {
    if (is_numeric($date)) {
        $timestamp = $date;
    } else {
        $timestamp = strtotime($date);
    }
    
    if ($format) {
        return date($format, $timestamp);
    }
    
    // Default to a nice readable format
    return date('F j, Y', $timestamp);
}

/**
 * Truncate a string to a maximum length, adding ellipsis if needed
 * 
 * @param string $text The text to truncate
 * @param int $length Maximum length
 * @param string $append String to append if truncated (default: ...)
 * @return string Truncated string
 */
function truncate($text, $length = 100, $append = '...') {
    if (strlen($text) <= $length) {
        return $text;
    }
    
    return substr($text, 0, $length) . $append;
}

/**
 * Highlight a search term in a text
 * 
 * @param string $text The text to search in
 * @param string $term The term to highlight
 * @param string $highlightClass CSS class for the highlight (default: highlight)
 * @return string Text with highlighted terms
 */
function highlight_term($text, $term, $highlightClass = 'highlight') {
    if (empty($term)) {
        return $text;
    }
    
    return preg_replace(
        '/(' . preg_quote($term, '/') . ')/i',
        '<span class="' . $highlightClass . '">$1</span>',
        $text
    );
}

/**
 * Check if the current page matches a given path
 * 
 * @param string $path The path to check against
 * @return bool True if current page matches the path
 */
function is_current_page($path) {
    $currentPath = $_SERVER['REQUEST_URI'] ?? '/';
    return ($currentPath == $path);
}

/**
 * Generate URL-friendly slug from a string
 * 
 * @param string $text The text to convert to a slug
 * @return string URL-friendly slug
 */
function slugify($text) {
    // Replace non-alphanumeric characters with hyphens
    $text = preg_replace('~[^\pL\d]+~u', '-', $text);
    // Transliterate
    $text = iconv('utf-8', 'us-ascii//TRANSLIT', $text);
    // Remove unwanted characters
    $text = preg_replace('~[^-\w]+~', '', $text);
    // Trim
    $text = trim($text, '-');
    // Remove duplicate hyphens
    $text = preg_replace('~-+~', '-', $text);
    // Convert to lowercase
    $text = strtolower($text);
    
    return $text;
} 