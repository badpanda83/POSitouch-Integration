$body = @{
    reference_number = "test-$(Get-Date -Format 'HHmmss')"
    table_number     = "999"
    cost_center      = "1"
    server_number    = "976"
    terminal_number  = "1"
    number_in_party  = 1
    items = @(
        @{
            screen_cell = "13,42"
            item_name   = "NAME"
            quantity    = 1
        }
    )
} | ConvertTo-Json -Depth 5

Write-Host "Sending order..." -ForegroundColor Cyan
Write-Host $body

$response = Invoke-RestMethod `
    -Uri "http://localhost:8080/api/v1/tickets" `
    -Method POST `
    -Body $body `
    -ContentType "application/json"

Write-Host "Response:" -ForegroundColor Green
$response