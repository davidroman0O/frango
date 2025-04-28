<?php
// XML response test without using SimpleXML
header('Content-Type: application/xml');

// Create XML manually
echo '<?xml version="1.0" encoding="UTF-8"?>
<response>
    <success>true</success>
    <message>This is an XML response from PHP</message>
    <timestamp>' . time() . '</timestamp>
    <data>
        <items>
            <item id="1">
                <name>Item 1</name>
                <price>19.99</price>
            </item>
            <item id="2">
                <name>Item 2</name>
                <price>29.99</price>
            </item>
            <item id="3">
                <name>Item 3</name>
                <price>39.99</price>
            </item>
        </items>
        <count>3</count>
        <page>1</page>
        <totalPages>1</totalPages>
    </data>
    <meta>
        <apiVersion>1.0</apiVersion>
        <serverTime>' . date('c') . '</serverTime>
    </meta>
</response>'; 