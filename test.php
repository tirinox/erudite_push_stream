<?php
error_reporting(E_ALL);
   $fp = fsockopen("localhost", 10026, $errno, $errstr, 30);
   if (!$fp) {
       echo "$errstr ($errno)\n";
   } else {
        $m = json_encode([
            'command' => 'publish',
            'api_key' => 'b14bf19bf0e48177eb2e',
            'ident' => 'erudite_123',
            'message' => 'Hello world! 123'
        ]);


       fwrite($fp, $m."\n");

       fflush($fp);
       while (!feof($fp)) {
           echo fgets($fp, 128);
       }
       fclose($fp);
   }