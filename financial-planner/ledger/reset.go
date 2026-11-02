package ledger

// Reset очищает хранилище в памяти для изоляции тестовых сценариев.
func Reset() {
	stateMu.Lock()
	defer stateMu.Unlock()

	transactions = make([]Transaction, 0)
	budgets = make(map[string]Budget)
}
