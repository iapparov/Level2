package app

import(
	"testing"
)

func TestEmptyString(t *testing.T){
	result, err := StrUnpack("")

	if err != nil{
		t.Errorf("Expected no err, got %v", err)
	}
	if result != ""{
		t.Fatalf("Expected empty str, got %s", result)
	}
}

func TestOnlyNumbers(t *testing.T){
	result, err := StrUnpack("12345")

	if err == nil{
		t.Errorf("Expected 'string contains only numbers', got nil")
	}
	if result != ""{
		t.Fatalf("Expected empty str, got %s", result)
	}
}

func TestNoDigits(t *testing.T){
	result, err := StrUnpack("abcd")

	if err != nil{
		t.Errorf("Expected no err, got %v", err)
	}
	if result != "abcd"{
		t.Fatalf("Expected 'abcd', got %s", result)
	}
}

func TestSimpleRepetition(t *testing.T){
	result, err := StrUnpack("a4bc2d5e")

	if err != nil{
		t.Errorf("Expected no err, got %v", err)
	}
	if result != "aaaabccddddde"{
		t.Fatalf("Expected 'aaaabccddddde', got %s", result)
	}
}

func TestEscaping(t *testing.T){
	result, err := StrUnpack("qwe\\4\\5")

	if err != nil{
		t.Errorf("Expected no err, got %v", err)
	}
	if result != "qwe45"{
		t.Fatalf("Expected 'qwe45', got %s", result)
	}
}

func TestEscapingWithRepetition(t *testing.T){
	result, err := StrUnpack("qwe\\45")

	if err != nil{
		t.Errorf("Expected no err, got %v", err)
	}
	if result != "qwe44444"{
		t.Fatalf("Expected 'qwe44444', got %s", result)
	}
}

func TestComplexString(t *testing.T){
	result, err := StrUnpack("qwe\\4\\5c3")

	if err != nil{
		t.Errorf("Expected no err, got %v", err)
	}
	if result != "qwe45ccc"{
		t.Fatalf("Expected 'qwe45ccc', got %s", result)
	}
}

func TestStringStartingWithDigit(t *testing.T){
	result, err := StrUnpack("3abc")

	if err != nil{
		t.Errorf("Expected no err, got %v", err)
	}
	if result != "abc"{
		t.Fatalf("Expected 'aaabc', got %s", result)
	}
}

func TestStringWithZeroRepetition(t *testing.T){
	result, err := StrUnpack("ab0c")

	if err == nil{
		t.Errorf("Expected 'digits must be grower than 0', got %v", err)
	}
	if result != ""{
		t.Fatalf("Expected ' ', got %s", result)
	}
}

func TestStringWithOnlyEscapedDigits(t *testing.T){
	result, err := StrUnpack("\\1\\2\\3")

	if err != nil{
		t.Errorf("Expected no err, got %v", err)
	}
	if result != "123"{
		t.Fatalf("Expected '123', got %s", result)
	}
}

func TestStringWithMixedEscapedAndUnescapedDigits(t *testing.T){
	result, err := StrUnpack("a2\\3b4\\5")

	if err != nil{
		t.Errorf("Expected no err, got %v", err)
	}
	if result != "aa3bbbb5"{
		t.Fatalf("Expected 'aa3bbbb5', got %s", result)
	}
}

func TestStringWithTrailingEscape(t *testing.T){
	result, err := StrUnpack("abc\\")

	if err != nil{
		t.Errorf("Expected no err, got %v", err)
	}
	if result != "abc"{
		t.Fatalf("Expected 'abc', got %s", result)
	}
}
