import re
with open('D:\\\Minibili\\\internal\\\handler\\\user_dynamic.go', 'r', encoding='utf-8') as f:
    c = f.read()

# Change 1: Replace user lookup at line ~91
c = c.replace(
    'var author model.User\n\t_ = a.DB.First(&author, dyn.UserID).Error',
    'var author model.User\n\tauthorPtr, _ := a.UserSvc.GetUserPublic(c.Request.Context(), dyn.UserID)\n\tif authorPtr != nil {\n\t\tauthor.ID = authorPtr.ID\n\t\tauthor.Username = authorPtr.Username\n\t\tauthor.AvatarURL = authorPtr.AvatarURL\n\t}'
)

# Change 2: Replace user lookup at line ~553
c = c.replace(
    'var u model.User\n\tif err := a.DB.First(&u, uid).Error; err != nil {\n\t\tresp.Err(c, http.StatusNotFound, errcode.CodeNotFound)\n\t\treturn\n\t}',
    'var u model.User\n\tuPtr, err := a.UserSvc.GetUserPublic(c.Request.Context(), uid)\n\tif err != nil || uPtr == nil {\n\t\tresp.Err(c, http.StatusNotFound, errcode.CodeNotFound)\n\t\treturn\n\t}\n\tu.ID = uPtr.ID\n\tu.Username = uPtr.Username\n\tu.AvatarURL = uPtr.AvatarURL'
)

with open('D:\\\Minibili\\\internal\\\handler\\\user_dynamic.go', 'w', encoding='utf-8') as f:
    f.write(c)
print('done')

