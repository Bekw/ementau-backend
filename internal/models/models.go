package models

import "time"

// UserRole maps to admin.user_role_tab.
type UserRole struct {
	UserRoleID   int    `db:"user_role_id"   json:"user_role_id"`
	UserRoleName string `db:"user_role_name" json:"user_role_name"`
	UserRoleCode string `db:"user_role_code" json:"user_role_code"`
}

// User maps to admin.user_tab.
type User struct {
	UserID     int       `db:"user_id"      json:"user_id"`
	FIO        string    `db:"fio"          json:"fio"`
	Email      string    `db:"email"        json:"email"`
	Password   string    `db:"password"     json:"-"`
	UserRoleID int       `db:"user_role_id" json:"user_role_id"`
	PhoneNum   string    `db:"phone_num"    json:"phone_num"`
	Rowversion time.Time `db:"rowversion"   json:"rowversion"`
}

// MaterialType maps to public.material_type_tab.
type MaterialType struct {
	MaterialTypeID   int    `db:"material_type_id"   json:"material_type_id"`
	MaterialTypeName string `db:"material_type_name" json:"material_type_name"`
	MaterialTypeCode string `db:"material_type_code" json:"material_type_code"`
}

// BuildingType maps to public.building_type_tab.
type BuildingType struct {
	BuildingTypeID   int    `db:"building_type_id"   json:"building_type_id"`
	BuildingTypeName string `db:"building_type_name" json:"building_type_name"`
	BuildingTypeCode string `db:"building_type_code" json:"building_type_code"`
}

// ProjectType maps to public.project_type_tab.
type ProjectType struct {
	ProjectTypeID   int    `db:"project_type_id"   json:"project_type_id"`
	ProjectTypeName string `db:"project_type_name" json:"project_type_name"`
	ProjectTypeCode string `db:"project_type_code" json:"project_type_code"`
}

// UnitType maps to public.unit_type_tab.
type UnitType struct {
	UnitTypeID   int    `db:"unit_type_id"   json:"unit_type_id"`
	UnitTypeName string `db:"unit_type_name" json:"unit_type_name"`
	UnitTypeCode string `db:"unit_type_code" json:"unit_type_code"`
}

// Material maps to public.material_tab.
type Material struct {
	MaterialID     int    `db:"material_id"      json:"material_id"`
	MaterialName   string `db:"material_name"    json:"material_name"`
	MaterialCode   string `db:"material_code"    json:"material_code"`
	MaterialTypeID int    `db:"material_type_id" json:"material_type_id"`
	PhotoName      string `db:"photo_name"       json:"photo_name"`
	PhotoURL       string `db:"photo_url"        json:"photo_url"`
	UnitTypeID     int    `db:"unit_type_id"     json:"unit_type_id"`
}

// ProjectFile maps to public.project_file_tab.
type ProjectFile struct {
	ProjectFileID int       `db:"project_file_id" json:"project_file_id"`
	ProjectID     int       `db:"project_id"      json:"project_id"`
	FileName      string    `db:"file_name"       json:"file_name"`
	FileURL       string    `db:"file_url"        json:"file_url"`
	EmployeeID    int       `db:"employee_id"     json:"employee_id"`
	Rowversion    time.Time `db:"rowversion"      json:"rowversion"`
}

// Supplier maps to public.supplier_tab.
type Supplier struct {
	SupplierID      int    `db:"supplier_id"      json:"supplier_id"`
	SupplierName    string `db:"supplier_name"    json:"supplier_name"`
	SupplierPhone   string `db:"supplier_phone"   json:"supplier_phone"`
	SupplierAddress string `db:"supplier_address" json:"supplier_address"`
}

// Project maps to public.project_tab.
type Project struct {
	ProjectID     int       `db:"project_id"      json:"project_id"`
	ProjectName   string    `db:"project_name"    json:"project_name"`
	ProjectTypeID int       `db:"project_type_id" json:"project_type_id"`
	EmployeeID    int       `db:"employee_id"     json:"employee_id"`
	Rowversion    time.Time `db:"rowversion"      json:"rowversion"`
}

// FinanceType maps to public.finance_type_tab.
type FinanceType struct {
	FinanceTypeID   int    `db:"finance_type_id"   json:"finance_type_id"`
	FinanceTypeName string `db:"finance_type_name" json:"finance_type_name"`
	FinanceTypeCode string `db:"finance_type_code" json:"finance_type_code"`
	IsInvest        bool   `db:"is_invest"         json:"is_invest"`
}

// Finance maps to public.finance_tab.
type Finance struct {
	FinanceID          int       `db:"finance_id"          json:"finance_id"`
	FinanceTypeID      int       `db:"finance_type_id"     json:"finance_type_id"`
	FinanceDescription string    `db:"finance_description" json:"finance_description"`
	FinanceFileName    string    `db:"finance_file_name"   json:"finance_file_name"`
	FinanceFileURL     string    `db:"finance_file_url"    json:"finance_file_url"`
	EmployeeID         int       `db:"employee_id"         json:"employee_id"`
	Rowversion         time.Time `db:"rowversion"          json:"rowversion"`
	ProjectID          int       `db:"project_id"          json:"project_id"`
	BuildingID         *int      `db:"building_id"         json:"building_id"`
	Amount             float64   `db:"amount"              json:"amount"`
}

// SupplyRequest maps to public.supply_request_tab.
type SupplyRequest struct {
	RequestID       int       `db:"request_id"  json:"request_id"`
	BuildingID      int       `db:"building_id" json:"building_id"`
	Status          string    `db:"status"      json:"status"` // created, sent, received
	EmployeeID      int       `db:"employee_id" json:"employee_id"`
	Comment         string    `db:"comment"     json:"comment"`
	Rowversion      time.Time `db:"rowversion"  json:"rowversion"`
	ParentRequestID *int      `db:"parent_request_id" json:"parent_request_id"`
}

// SupplyRequestItem maps to public.supply_request_item_tab.
type SupplyRequestItem struct {
	ItemID           int      `db:"item_id"           json:"item_id"`
	RequestID        int      `db:"request_id"        json:"request_id"`
	MaterialID       int      `db:"material_id"       json:"material_id"`
	Quantity         float64  `db:"quantity"          json:"quantity"`
	Comment          string   `db:"comment"           json:"comment"`
	Price            float64  `db:"price"             json:"price"`
	ReceivedStatus   *string  `db:"received_status"   json:"received_status"`
	ReceivedQuantity *float64 `db:"received_quantity" json:"received_quantity"`
	SupplierID       *int     `db:"supplier_id"       json:"supplier_id"`
	UnitTypeID       *int     `db:"unit_type_id"      json:"unit_type_id"`
}

// Building maps to public.building_tab.
type Building struct {
	BuildingID      int        `db:"building_id"      json:"building_id"`
	BuildingName    string     `db:"building_name"    json:"building_name"`
	BuildingTypeID  int        `db:"building_type_id" json:"building_type_id"`
	BuildingAddress string     `db:"building_address" json:"building_address"`
	EmployeeID      int        `db:"employee_id"      json:"employee_id"`
	DateStart       *time.Time `db:"date_start"       json:"date_start"`
	DateEnd         *time.Time `db:"date_end"         json:"date_end"`
	Rowversion      time.Time  `db:"rowversion"       json:"rowversion"`
	ProjectID       int        `db:"project_id"       json:"project_id"`
}
